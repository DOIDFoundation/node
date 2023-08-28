package network

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/version"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	dsbadger "github.com/ipfs/go-ds-badger"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

var ctx, cancelCtx = context.WithCancel(context.Background())

type Network struct {
	service.BaseService

	host       host.Host
	discovery  *discovery
	pubsub     *pubsub.PubSub
	topicBlock *pubsub.Topic
	topicTx    *pubsub.Topic
	blockChain *core.BlockChain
	syncing    atomic.Bool
	sync       *syncService
}

func NewNetwork(chain *core.BlockChain, logger log.Logger) *Network {
	initConstants()
	network := &Network{
		blockChain: chain,
	}
	network.BaseService = *service.NewBaseService(logger.With("module", "network"), "Network", network)

	dataDir := filepath.Join(viper.GetString(flags.Home), "data")

	if fi, err := os.Stat(dataDir); err == nil {
		if !fi.IsDir() {
			network.Logger.Error("not a directory", "path", dataDir)
			return nil
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			network.Logger.Error("failed to create directory", "path", dataDir, "err", err)
			return nil
		}
	} else {
		network.Logger.Error("failed to check data directory", "path", dataDir, "err", err)
		return nil
	}

	ds, err := dsbadger.NewDatastore(filepath.Join(dataDir, "libp2p-peerstore-v0"), &dsbadger.DefaultOptions)
	if err != nil {
		network.Logger.Error("Failed to create badger store", "err", err, "path", filepath.Join(dataDir, "libp2p-peerstore-v0"))
		return nil
	}

	ps, err := pstoreds.NewPeerstore(ctx, ds, pstoreds.DefaultOpts())
	if err != nil {
		network.Logger.Error("Failed to create peerstore", "err", err)
		return nil
	}

	dsDht, err := dsbadger.NewDatastore(filepath.Join(dataDir, "libp2p-dht-v0"), &dsbadger.DefaultOptions)
	if err != nil {
		network.Logger.Error("Failed to create dht store", "err", err, "path", filepath.Join(dataDir, "libp2p-dht-v0"))
		return nil
	}

	// Start with the default scaling limits.
	scalingLimits := rcmgr.DefaultLimits

	// Add limits around included libp2p protocols
	libp2p.SetDefaultServiceLimits(&scalingLimits)

	// Turn the scaling limits into a concrete set of limits using `.AutoScale`. This
	// scales the limits proportional to your system memory.
	scaledDefaultLimits := scalingLimits.AutoScale()

	// Tweak certain settings
	cfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			// Allow unlimited outbound streams
			StreamsOutbound: rcmgr.Unlimited,
		},
		// Everything else is default. The exact values will come from `scaledDefaultLimits` above.
	}

	// Create our limits by using our cfg and replacing the default values with values from `scaledDefaultLimits`
	limits := cfg.Build(scaledDefaultLimits)

	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(limits)

	// Initialize the resource manager
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		network.Logger.Error("Failed to create libp2p resource manager", "err", err)
		return nil
	}

	cm, err := connmgr.NewConnManager(20, 50)
	if err != nil {
		network.Logger.Error("Failed to create libp2p connection manager", "err", err)
		return nil
	}

	var idht *dual.DHT
	options := []config.Option{
		libp2p.Identity(network.loadPrivateKey()),
		libp2p.ListenAddrStrings(viper.GetStringSlice(flags.P2P_Addr)...),
		libp2p.UserAgent(UserAgent()),
		libp2p.ResourceManager(rm),
		libp2p.ConnectionManager(cm),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Peerstore(ps),
		libp2p.EnableNATService(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dual.New(ctx, h,
				dual.WanDHTOption(dht.Datastore(dsDht)),
				dual.DHTOption(dht.BootstrapPeers(network.discovery.bootstrapPeers()...)),
				dual.DHTOption(dht.ProtocolPrefix("/doid/kad/1")),
			)
			return idht, err
		}),
		libp2p.AddrsFactory(network.addrsFactory),
	}
	if viper.GetBool(flags.P2P_RelaySvc) {
		options = append(options, libp2p.EnableRelayService())
	}
	if viper.GetBool(flags.P2P_uPNP) {
		// NATPortMap will attempt to open ports using uPNP for NATed hosts.
		options = append(options, libp2p.NATPortMap())
	}
	switch viper.GetString(flags.P2P_Reachability) {
	case "public":
		options = append(options, libp2p.ForceReachabilityPublic())
	case "private":
		options = append(options, libp2p.ForceReachabilityPrivate())
	}
	network.host, err = libp2p.New(options...)
	if err != nil {
		network.Logger.Error("Failed to create libp2p host", "err", err)
		return nil
	}
	network.Logger.Info("Host created.", "id", network.host.ID(), "addrs", network.host.Addrs())

	network.pubsub, err = pubsub.NewGossipSub(ctx, network.host)
	if err != nil {
		network.Logger.Error("Failed to create pubsub", "err", err)
		return nil
	}

	network.discovery = newDiscovery(logger, chain, network.host, idht, network.pubsub)
	if network.discovery == nil {
		network.Logger.Error("Failed to create discovery service")
		return nil
	}

	network.registerEventHandlers()
	RegisterAPI(network)
	return network
}

// OnStart starts the Network. It implements service.Service.
func (n *Network) OnStart() error {
	n.host.Network().Notify(n)
	n.host.SetStreamHandler(protocol.ID(ProtocolGetBlocks), n.getBlocksHandler)
	n.host.SetStreamHandler(protocol.ID(ProtocolState), n.stateHandler)

	go n.joinTopicBlock()
	go n.joinTopicTx()

	n.discovery.Start()

	return nil
}

// OnStop stops the Network. It implements service.Service.
func (n *Network) OnStop() {
	n.host.Network().StopNotify(n)
	cancelCtx()
	n.stopSync()
	n.discovery.Stop()
	n.host.Close()
	n.unregisterEventHandlers()
}

func (n *Network) startSync() {
	if !n.syncing.CompareAndSwap(false, true) {
		n.Logger.Debug("not starting sync", "msg", "already started")
		return
	}
	// find a best peer with most total difficulty
	var best *state
	var bestId peer.ID
	peerHasState.Range(func(key, _ interface{}) bool {
		id, ok := key.(peer.ID)
		if !ok {
			n.Logger.Error("failed to convert key to peer id", "key", key)
			return true
		}
		peerState := getPeerState(n.host.Peerstore(), id)
		if best == nil || (peerState != nil && peerState.Td.Cmp(best.Td) > 0) {
			best = peerState
			bestId = id
		}
		return true
	})
	if best == nil || best.Td.Cmp(n.blockChain.GetTd()) <= 0 {
		n.Logger.Debug("not starting sync", "msg", "no better network td")
		n.syncing.Store(false)
		return
	}
	n.Logger.Info("start syncing")
	events.SyncStarted.Send()
	events.SyncFinished.Subscribe(n.String(), func() { n.stopSync() })
	eventSyncFailed.Subscribe(n.String(), func() { n.stopSync(); n.startSync() })
	n.sync = newSyncService(n.Logger, bestId, n.host, n.blockChain)
	n.sync.Start()
}

func (n *Network) stopSync() {
	events.SyncFinished.Unsubscribe(n.String())
	eventSyncFailed.Unsubscribe(n.String())
	if !n.syncing.CompareAndSwap(true, false) {
		n.Logger.Debug("not stopping sync", "msg", "already stopped")
		return
	}
	n.sync.Stop()
	n.sync = nil
}

func (n *Network) unregisterEventHandlers() {
	eventPeerState.Unsubscribe(n.String())
	events.ForkDetected.Unsubscribe(n.String())
	events.NewMinedBlock.Unsubscribe(n.String())
	events.NewTx.Unsubscribe(n.String())
}

func (n *Network) registerEventHandlers() {
	var once sync.Once
	eventPeerState.Subscribe(n.String(), func(pid peer.ID) {
		peerState := getPeerState(n.host.Peerstore(), pid)
		n.Logger.Debug("got peer state", "peer", pid, "state", peerState)
		if peerState != nil {
			switch peerState.Td.Cmp(n.blockChain.GetTd()) {
			case -1: // we are high
				go func() {
					ctx, cancel := context.WithTimeout(ctx, time.Second*15)
					defer cancel()
					stream, err := n.host.NewStream(ctx, pid, protocol.ID(ProtocolState))
					if err != nil {
						n.Logger.Debug("failed to create stream", "err", err, "peer", pid)
						return
					}
					stream.CloseRead()
					n.stateHandler(stream)
				}()
			case 0:
				// try connect if we do not have enough peers
				if len(n.host.Network().Peers()) < MinConnectedPeers {
					go func() {
						if err := n.host.Connect(ctx, peer.AddrInfo{ID: pid}); err != nil {
							n.Logger.Debug("failed to connect", "peer", pid, "err", err)
						}
					}()
				}
			case 1: // network is high
				n.startSync()
			}
			// Send a sync finished event only once for the first time when a
			// peer connected but we do not need to sync
			if n.sync == nil || !n.sync.IsRunning() {
				once.Do(func() {
					events.SyncFinished.Send()
				})
			}
		}
	})
	events.ForkDetected.Subscribe(n.String(), func() {
		n.startSync()
	})
	events.NewMinedBlock.Subscribe(n.String(), func(data *types.Block) {
		if n.topicBlock == nil {
			n.Logger.Info("not broadcasting, new block topic not joined")
			return
		}
		b, err := rlp.EncodeToBytes(events.BlockWithTd{Block: data, Td: n.blockChain.GetTd()})
		if err != nil {
			n.Logger.Error("failed to encode block for broadcasting", "err", err)
			return
		}
		n.topicBlock.Publish(ctx, b)
	})
	events.NewTx.Subscribe(n.String(), func(data types.Tx) {
		if n.topicTx == nil {
			n.Logger.Info("not broadcasting, new tx topic not joined")
			return
		}
		b, err := rlp.EncodeToBytes(data)
		if err != nil {
			n.Logger.Error("failed to encode tx for broadcasting", "err", err)
			return
		}
		n.topicTx.Publish(ctx, b)
	})
}

func (n *Network) loadPrivateKey() crypto.PrivKey {
	// try base64 key flag
	if viper.IsSet(flags.P2P_Key) {
		priv, err := decodeKey(viper.GetString(flags.P2P_Key))
		if err != nil {
			n.Logger.Error("Failed to parse p2p key", "err", err)
			panic(err)
		}
		return priv
	}

	var priv crypto.PrivKey
	// try key file first
	keyFile := filepath.Join(viper.GetString(flags.Home), viper.GetString(flags.P2P_KeyFile))
	bz, err := os.ReadFile(keyFile)
	logger := n.Logger.With("file", keyFile)
	if err == nil {
		priv, err = decodeKey(string(bz))
		if err != nil {
			logger.Error("Failed to parse p2p key file", "err", err)
			panic(err)
		}
	} else if os.IsNotExist(err) && viper.GetString(flags.P2P_KeyFile) == "p2p.key" {
		priv, _, err = crypto.GenerateKeyPair(
			crypto.Ed25519, // Select your key type. Ed25519 are nice short
			-1,             // Select key length when possible (i.e. RSA).
		)
		if err != nil {
			logger.Error("Failed to generate p2p key", "err", err)
			panic(err)
		}
		bz, err = priv.Raw()
		if err == nil {
			err = os.WriteFile(keyFile, []byte(crypto.ConfigEncodeKey(bz)), 0600)
		}
		if err != nil {
			logger.Error("Failed to write p2p key bytes", "err", err)
		}
	} else {
		logger.Error("Failed to read p2p key file", "err", err)
		panic(err)
	}

	return priv
}

func decodeKey(s string) (crypto.PrivKey, error) {
	bz, err := crypto.ConfigDecodeKey(s)
	if err != nil {
		return nil, err
	} else {
		return crypto.UnmarshalEd25519PrivateKey(bz)
	}
}

func UserAgent() string {
	vsn := "doid/"
	if version.VersionMeta != "stable" {
		vsn += version.VersionWithMeta
	} else {
		vsn += version.Version
	}
	if len(version.Commit) >= 8 {
		vsn += "/" + version.Commit[:8]
	}
	if (version.VersionMeta != "stable") && (version.Date != "") {
		vsn += "/" + version.Date
	}
	return vsn
}

var ErrInvalidFormat = errors.New("invalid multiaddr-filter format")

func NewMask(a string) (*net.IPNet, error) {
	parts := strings.Split(a, "/")

	if parts[0] != "" {
		return nil, ErrInvalidFormat
	}

	if len(parts) != 5 {
		return nil, ErrInvalidFormat
	}

	// check it's a valid filter address. ip + cidr
	isip := parts[1] == "ip4" || parts[1] == "ip6"
	iscidr := parts[3] == "ipcidr"
	if !isip || !iscidr {
		return nil, ErrInvalidFormat
	}

	_, ipn, err := net.ParseCIDR(parts[2] + "/" + parts[4])
	if err != nil {
		return nil, err
	}
	return ipn, nil
}

func (n *Network) addrsFactory(as []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	parseAddrs := func(addrs []string, ret *[]multiaddr.Multiaddr) {
		for _, addr := range addrs {
			a, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				n.Logger.Error("Failed to parse address", "err", err, "addr", addr)
				continue
			}
			*ret = append(*ret, a)
		}
	}
	var addrs []multiaddr.Multiaddr
	if ann := viper.GetStringSlice(flags.P2P_AnnAddr); len(ann) != 0 {
		parseAddrs(ann, &addrs)
	} else {
		addrs = as
	}
	if add := viper.GetStringSlice(flags.P2P_AnnAddrAppend); len(add) != 0 {
		parseAddrs(add, &addrs)
	}
	if exclude := viper.GetStringSlice(flags.P2P_AnnAddrRemove); len(exclude) != 0 {
		filters := multiaddr.NewFilters()
		noAnnAddrs := map[string]bool{}
		for _, addr := range exclude {
			f, err := NewMask(addr)
			if err == nil {
				filters.AddFilter(*f, multiaddr.ActionDeny)
				continue
			}
			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				n.Logger.Error("Failed to parse address", "err", err, "addr", addr)
				continue
			}
			noAnnAddrs[string(maddr.Bytes())] = true
		}
		var out []multiaddr.Multiaddr
		for _, maddr := range addrs {
			// check for exact matches
			ok := noAnnAddrs[string(maddr.Bytes())]
			// check for /ipcidr matches
			if !ok && !filters.AddrBlocked(maddr) {
				out = append(out, maddr)
			}
		}
		return out
	}

	return addrs
}
