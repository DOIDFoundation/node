package network

import (
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-datastore"
	dsbadger "github.com/ipfs/go-ds-badger"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

// discovery gets notified when we find a new peer via mDNS discovery
type discovery struct {
	service.BaseService
	chain  *core.BlockChain
	host   host.Host
	pubsub *pubsub.PubSub
	dht    *dual.DHT
	ds     *dsbadger.Datastore

	mdns mdns.Service
}

func newDiscovery(logger log.Logger, chain *core.BlockChain, h host.Host, dht *dual.DHT, pubsub *pubsub.PubSub) *discovery {
	dataDir := filepath.Join(viper.GetString(flags.Home), "data", "libp2p-peers-backup")

	ds, err := dsbadger.NewDatastore(dataDir, &dsbadger.DefaultOptions)
	if err != nil {
		logger.Error("Failed to create badger store", "err", err, "path", dataDir)
		return nil
	}
	d := &discovery{chain: chain, host: h, pubsub: pubsub, dht: dht, ds: ds}
	d.BaseService = *service.NewBaseService(logger.With("module", "network", "service", "discovery"), "Discovery", d)
	// setup mDNS discovery to find local peers
	d.mdns = mdns.NewMdnsService(h, viper.GetString(flags.P2P_Rendezvous), d)
	return d
}

func (d *discovery) OnStart() error {
	go d.setupDiscover()
	return nil
}

func (d *discovery) OnStop() {
	if err := d.ds.Close(); err != nil {
		d.Logger.Error("failed to close datastore", "err", err)
	}
	if err := d.mdns.Close(); err != nil {
		d.Logger.Error("failed to close mdns discovery", "err", err)
	}
}

func (d *discovery) pubsubDiscover() {
	logger := d.Logger.With("topic", TopicPeer)
	topic, err := d.pubsub.Join(TopicPeer)
	if err != nil {
		logger.Error("Failed to join pubsub topic", "err", err)
		return
	}
	logger.Debug("pubsub topic joined")

	peerFound := func(msg *pubsub.Message) {
		data := msg.GetData()
		peerState := newState()
		if err := rlp.DecodeBytes(data, peerState); err != nil {
			logger.Error("failed to decode peer state", "err", err)
			return
		}
		peer := msg.GetFrom()
		if updated, err := updatePeerState(d.host.Peerstore(), peer, peerState); updated == true {
			logger.Debug("pubsub found peer", "peer", peer, "state", peerState)
			eventPeerState.Send(peer)
		} else if err != nil {
			logger.Error("failed to update peer state", "peer", peer, "err", err)
		}
	}

	go pubsubMessageLoop(ctx, topic, d.host.ID(), peerFound, logger)

	duration := time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		if len(topic.ListPeers()) == 0 {
			if duration != time.Second {
				duration = time.Second
				ticker.Reset(duration)
			}
			logger.Debug("no peer in topic, wait")
		} else if bz, err := rlp.EncodeToBytes(&state{Height: d.chain.LatestBlock().Header.Height.Uint64(), Td: new(big.Int).Set(d.chain.GetTd())}); err != nil {
			logger.Error("failed to encode peer state", "err", err)
		} else if err = topic.Publish(ctx, bz); err != nil {
			logger.Error("failed to publish peer state", "err", err)
		} else {
			if duration != time.Minute {
				duration = time.Minute
				ticker.Reset(duration)
			}
			logger.Debug("publish our state")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			continue
		}
	}
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (d *discovery) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID.String() == d.host.ID().String() {
		return
	}
	d.Logger.Debug("mdns found peer", "peer", pi)
	peerNotifier <- pi
}

func (n *Network) notifyPeerFoundEvent() {
	for {
		pi := <-peerNotifier
		switch n.host.Network().Connectedness(pi.ID) {
		case network.Connected:
			n.Logger.Debug("already connected", "peer", pi)
			continue
		case network.CannotConnect:
			n.Logger.Debug("not connectable", "peer", pi)
			continue
		}

		stream, err := n.host.NewStream(ctx, pi.ID, protocol.ID(ProtocolState))
		if err != nil {
			n.Logger.Debug("failed to create stream", "err", err, "peer", pi)
			continue
		}
		n.Logger.Debug("stream established", "peer", pi)
		n.stateHandler(stream)
		stream.Close()
	}
}

func (d *discovery) bootstrapPeers() (addrs []*peer.AddrInfo) {
	for _, peerAddr := range dht.DefaultBootstrapPeers { // @todo add a flag/config
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			d.Logger.Error("failed to parse bootstrap peer", "peer", peerAddr, "err", err)
			continue
		}
		addrs = append(addrs, peerInfo)
	}
	return
}

func (d *discovery) connect(peerInfo peer.AddrInfo) {
	d.Logger.Debug("try connect", "peer", peerInfo)
	if err := d.host.Connect(ctx, peerInfo); err != nil {
		d.Logger.Error("failed to connect", "peer", peerInfo, "err", err)
	} else {
		d.Logger.Info("connected", "peer", peerInfo)
	}
}

func (d *discovery) setupDiscover() {
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	BootstrapPeers := d.bootstrapPeers()
	var wg sync.WaitGroup
	for _, peerInfo := range BootstrapPeers {
		wg.Add(1)
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			d.connect(*peerInfo)
		}(peerInfo)
	}

	// Now connect to the backup peers to speedup bootstrap.
	for _, peerInfo := range d.loadBackupPeers() {
		go d.connect(peerInfo)
	}

	// wait all bootstrap peers are connected or unreachable
	wg.Wait()

	// save connected peers
	go d.saveConnectedPeers()

	// start discovery after bootstrap
	d.Logger.Info("Discovering p2p network")

	if err := d.mdns.Start(); err != nil {
		d.Logger.Error("failed to start mdns discovery", "err", err)
	}

	go d.pubsubDiscover()

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	d.Logger.Debug("Bootstrapping DHT")
	if err := d.dht.Bootstrap(ctx); err != nil {
		d.Logger.Error("failed to bootstrap dht", "err", err)
		return
	}
	time.Sleep(time.Millisecond * 100)

	var routingDiscovery = routing.NewRoutingDiscovery(d.dht)
	if routingDiscovery == nil {
		d.Logger.Error("failed to create routing discovery")
		return
	}

	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	rendezvous := viper.GetString(flags.P2P_Rendezvous)
	for {
		if _, err := routingDiscovery.Advertise(ctx, rendezvous); err != nil {
			d.Logger.Error("failed to routing advertise: ", "err", err)
		} else if peers, err := routingDiscovery.FindPeers(ctx, rendezvous); err != nil {
			d.Logger.Error("failed to find peers", "err", err)
		} else {
			for p := range peers {
				if p.ID == d.host.ID() || len(p.Addrs) == 0 {
					continue
				}
				d.Logger.Debug("dht found peer", "peer", p)
				peerNotifier <- p
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			continue
		}
	}
}

func randomizeList[T any](in []T) []T {
	out := make([]T, len(in))
	for i, val := range rand.Perm(len(in)) {
		out[i] = in[val]
	}
	return out
}

var keyBackupPeers = datastore.NewKey("backup_peers")

func (d *discovery) loadBackupPeers() []peer.AddrInfo {
	var addrs []string
	bytes, err := d.ds.Get(ctx, keyBackupPeers)
	if err != nil {
		if !errors.Is(err, datastore.ErrNotFound) {
			d.Logger.Error("failed to load backup peers", "err", err)
		}
		return nil
	}
	if err := json.Unmarshal(bytes, &addrs); err != nil {
		d.Logger.Error("failed to parse backup peers", "err", err)
		return nil
	}

	maddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			d.Logger.Error("failed to parse backup peer", "err", err, "addr", addr)
			continue
		}
	}
	backupPeers, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		d.Logger.Error("failed to parse backup peers", "err", err)
		return nil
	}
	return backupPeers
}

func (d *discovery) saveBackupPeers(backupPeers []peer.AddrInfo) {
	bpss := make([]string, 0, len(backupPeers))
	for _, pi := range backupPeers {
		addrs, err := peer.AddrInfoToP2pAddrs(&pi)
		if err != nil {
			// programmer error.
			panic(err)
		}
		for _, addr := range addrs {
			bpss = append(bpss, addr.String())
		}
	}
	bytes, err := json.Marshal(bpss)
	if err != nil {
		d.Logger.Error("failed to save backup peers", "err", err)
		return
	}
	if err := d.ds.Put(ctx, keyBackupPeers, bytes); err != nil {
		d.Logger.Error("failed to save backup peers", "err", err)
		return
	}
	if err := d.ds.Sync(ctx, keyBackupPeers); err != nil {
		d.Logger.Error("failed to save backup peers", "err", err)
	}
}

func (d *discovery) saveConnectedPeers() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		MaxBackupBootstrapSize := 20
		// Randomize the list of connected peers, we don't prioritize anyone.
		connectedPeers := randomizeList(d.host.Network().Peers())
		bootstrapPeers := d.bootstrapPeers()
		backupPeers := make([]peer.AddrInfo, 0, MaxBackupBootstrapSize)

		// Choose peers to save and filter out the ones that are already bootstrap nodes.
		for _, p := range connectedPeers {
			found := false
			for _, bootstrapPeer := range bootstrapPeers {
				if p == bootstrapPeer.ID {
					found = true
					break
				}
			}
			if !found {
				backupPeers = append(backupPeers, peer.AddrInfo{
					ID:    p,
					Addrs: d.host.Network().Peerstore().Addrs(p),
				})
			}

			if len(backupPeers) >= MaxBackupBootstrapSize {
				break
			}
		}
		// If we didn't reach the target number use previously stored connected peers.
		if len(backupPeers) < MaxBackupBootstrapSize {
			oldSavedPeers := d.loadBackupPeers()
			d.Logger.Debug("not enough backup peers", "missing", MaxBackupBootstrapSize-len(backupPeers), "target", MaxBackupBootstrapSize, "saved", len(oldSavedPeers))

			// Add some of the old saved peers. Ensure we don't duplicate them.
			for _, p := range oldSavedPeers {
				found := false
				for _, sp := range backupPeers {
					if p.ID == sp.ID {
						found = true
						break
					}
				}

				if !found {
					backupPeers = append(backupPeers, p)
				}

				if len(backupPeers) >= MaxBackupBootstrapSize {
					break
				}
			}
		}

		d.saveBackupPeers(backupPeers)
		d.Logger.Debug("backup peers saved", "saved", len(backupPeers), "target", MaxBackupBootstrapSize)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			continue
		}
	}
}
