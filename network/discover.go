package network

import (
	"math/big"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/spf13/viper"
)

// discovery gets notified when we find a new peer via mDNS discovery
type discovery struct {
	service.BaseService
	chain  *core.BlockChain
	host   host.Host
	pubsub *pubsub.PubSub
	dht    *dual.DHT

	mdns mdns.Service
}

func newDiscovery(logger log.Logger, chain *core.BlockChain, h host.Host, dht *dual.DHT, pubsub *pubsub.PubSub) *discovery {
	d := &discovery{chain: chain, host: h, pubsub: pubsub, dht: dht}
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
		n.Logger.Debug("connected with peer", "peer", pi)
		n.stateHandler(stream)
		stream.Close()
	}
}

func (d *discovery) setupDiscover() {
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	BootstrapPeers := dht.DefaultBootstrapPeers // @todo add a flag/config
	var wg sync.WaitGroup
	for _, peerAddr := range BootstrapPeers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			d.Logger.Error("Failed to parse bootstrap peer", "peer", peerAddr, "err", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := d.host.Connect(ctx, *peerInfo); err != nil {
				d.Logger.Error("Failed to connect", "peer", *peerInfo, "err", err)
			} else {
				d.Logger.Info("Connection established with bootstrap network:", "peer", *peerInfo)
			}
		}()
	}
	wg.Wait()

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
