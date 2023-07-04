package network

import (
	"context"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/version"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

// ------------------------------------------------------------------------------
var peerPool = make(map[string]peer.AddrInfo)
var ctx = context.Background()
var peerNotifier = make(chan peer.AddrInfo)

type Network struct {
	service.BaseService
	config *Config

	host          host.Host
	discovery     *discovery
	pubsub        *pubsub.PubSub
	topicBlock    *pubsub.Topic
	blockChain    *core.BlockChain
	networkHeight *big.Int
	syncing       atomic.Bool
	sync          *syncService
	peerPool      map[string]peer.AddrInfo
}

// Option sets a parameter for the network.
type Option func(*Network)

// NewNetwork returns a new, ready to go, CometBFT Node.
func NewNetwork(chain *core.BlockChain, logger log.Logger) *Network {
	network := &Network{
		config:     &DefaultConfig,
		blockChain: chain,

		networkHeight: big.NewInt(0),
		peerPool:      make(map[string]peer.AddrInfo),
	}
	network.BaseService = *service.NewBaseService(logger.With("module", "network"), "Network", network)

	addr, err := multiaddr.NewMultiaddr(viper.GetString(flags.P2P_Addr))
	if err != nil {
		network.Logger.Error("Failed to parse p2p.addr", "err", err, "addr", viper.GetString(flags.P2P_Addr))
		return nil
	}

	// var idht *dht.IpfsDHT
	network.host, err = libp2p.New(
		libp2p.Identity(network.loadPrivateKey()),
		libp2p.ListenAddrs(addr),
		libp2p.UserAgent(version.VersionWithCommit()),
		libp2p.Security(noise.ID, noise.New),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err := dht.New(ctx, h)
			return idht, err
		}),
	)
	if err != nil {
		network.Logger.Error("Failed to create libp2p host", "err", err)
		return nil
	}
	network.Logger.Info("Host created.", "id", network.host.ID(), "addrs", network.host.Addrs())

	network.discovery = NewDiscovery(logger, network.host)
	if network.discovery == nil {
		network.Logger.Error("Failed to create discovery service")
		return nil
	}

	network.pubsub, err = pubsub.NewGossipSub(ctx, network.host)
	if err != nil {
		network.Logger.Error("Failed to create pubsub", "err", err)
		return nil
	}

	network.registerEventHandlers()
	RegisterAPI(network)
	return network
}

// OnStart starts the Network. It implements service.Service.
func (n *Network) OnStart() error {
	localHost := n.host

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	localHost.SetStreamHandler(protocol.ID(ProtocolID), n.handleStream)

	//_ = viper.GetString(node.config.BootstrapPeers)
	if len(n.config.BootstrapPeers) == 0 {
		n.config.BootstrapPeers = dht.DefaultBootstrapPeers
	}

	n.discovery.Start()

	go n.Bootstrap(localHost)
	localHost.SetStreamHandler(protocol.ID(ProtocolGetBlock), n.getBlockHandler)

	return nil
}

// OnStop stops the Network. It implements service.Service.
func (n *Network) OnStop() {
	n.stopSync()
	n.discovery.Stop()
	n.host.Close()
	events.ForkDetected.Unsubscribe(n.String())
	events.NewMinedBlock.Unsubscribe(n.String())
}

func (n *Network) startSync() {
	// find a best peer with most total difficulty
	var best *peerState
	for _, id := range n.host.Peerstore().Peers() {
		v, err := n.host.Peerstore().Get(id, metaVersion)
		if err != nil {
			continue
		}
		version := v.(peerState)
		if best == nil || version.Td.Cmp(best.Td) > 0 {
			best = &version
		}
	}
	if best == nil || best.Td.Cmp(n.blockChain.GetTd()) <= 0 {
		n.Logger.Debug("not starting sync", "msg", "no better network td")
		return
	}
	if !n.syncing.CompareAndSwap(false, true) {
		n.Logger.Debug("not starting sync", "msg", "already started")
		return
	}
	n.Logger.Info("start syncing")
	events.SyncStarted.Send(struct{}{})
	events.SyncFinished.Subscribe(n.String(), func(data struct{}) { n.stopSync() })
	n.sync = newSyncService(n.Logger, peerIDFromString(best.ID), n.host, n.blockChain)
	n.sync.Start()
}

func (n *Network) stopSync() {
	events.SyncFinished.Unsubscribe(n.String())
	if !n.syncing.CompareAndSwap(true, false) {
		n.Logger.Debug("not stopping sync", "msg", "already stopped")
		return
	}
	n.sync.Stop()
	n.sync = nil
}

func (n *Network) registerEventHandlers() {
	events.ForkDetected.Subscribe(n.String(), func(data struct{}) {
		n.startSync()
	})
	events.NewMinedBlock.Subscribe(n.String(), func(data *types.Block) {
		if n.topicBlock == nil {
			n.Logger.Info("not broadcasting, new block topic not joined")
			return
		}
		b, err := rlp.EncodeToBytes(newBlock{Block: data, Td: n.blockChain.GetTd()})
		if err != nil {
			n.Logger.Error("failed to encode block for broadcasting", "err", err)
			return
		}
		n.Logger.Debug("topic peers", "peers", n.topicBlock.ListPeers())
		n.topicBlock.Publish(ctx, b)
	})
}

func (n *Network) Bootstrap(newNode host.Host) {
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range n.config.BootstrapPeers {
		peerinfo2, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := newNode.Connect(ctx, *peerinfo2); err != nil {
				n.Logger.Error("Failed to connect", "peer", *peerinfo2, "err", err)
			} else {
				n.Logger.Info("Connection established with bootstrap network:", "peer", *peerinfo2)
			}
		}()
	}
	wg.Wait()

	//service := NewService(newNode, protocol.ID(ProtocolID), n.blockChain, n.Logger)
	//err := service.SetupRPC()
	//if err != nil {
	//	n.Logger.Error("new RPC service", "err", err)
	//}

	//n.rpcService = service

	go n.notifyPeerFoundEvent()
	go n.registerBlockSubscribers()
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
	} else if errors.Is(err, os.ErrNotExist) && viper.GetString(flags.P2P_KeyFile) == "p2p.key" {
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
