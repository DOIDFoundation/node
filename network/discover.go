package network

import (
	"sync"
	"time"

	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/spf13/viper"
)

// discovery gets notified when we find a new peer via mDNS discovery
type discovery struct {
	service.BaseService
	h    host.Host
	dht  *dual.DHT
	mdns mdns.Service
}

func NewDiscovery(logger log.Logger, h host.Host, dht *dual.DHT) *discovery {
	d := &discovery{h: h, dht: dht}
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

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (d *discovery) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID.String() == d.h.ID().String() {
		return
	}
	d.Logger.Debug("mdns found peer", "peer", pi)
	peerNotifier <- pi
}

func (n *Network) notifyPeerFoundEvent() {
	for {
		pi := <-peerNotifier

		stream, err := n.host.NewStream(ctx, pi.ID, protocol.ID(ProtocolState))
		if err != nil {
			n.Logger.Debug("failed to create stream", "err", err, "peer", pi)
			continue
		}
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
			if err := d.h.Connect(ctx, *peerInfo); err != nil {
				d.Logger.Error("Failed to connect", "peer", *peerInfo, "err", err)
			} else {
				d.Logger.Info("Connection established with bootstrap network:", "peer", *peerInfo)
			}
		}()
	}
	wg.Wait()

	if err := d.mdns.Start(); err != nil {
		d.Logger.Error("failed to start mdns discovery", "err", err)
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	d.Logger.Debug("Bootstrapping the DHT")
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
	rendezvous := viper.GetString(flags.P2P_Rendezvous)
	if _, err := routingDiscovery.Advertise(ctx, rendezvous); err != nil {
		d.Logger.Error("failed to routing advertise: ", "err", err)
		return
	}

	peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		d.Logger.Error("failed to find peers", "err", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-peers:
			if p.ID == d.h.ID() || len(p.Addrs) == 0 {
				continue
			}
			d.Logger.Debug("dht found peer", "peer", p)
			peerNotifier <- p
		}
	}
}
