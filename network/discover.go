package network

import (
	"context"
	"time"

	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/spf13/viper"
)

// discovery gets notified when we find a new peer via mDNS discovery
type discovery struct {
	service.BaseService
	h host.Host
	s mdns.Service
}

func NewDiscovery(logger log.Logger, h host.Host) *discovery {
	d := &discovery{h: h}
	d.BaseService = *service.NewBaseService(logger.With("service", "discovery"), "Discovery", d)
	// setup mDNS discovery to find local peers
	d.s = mdns.NewMdnsService(h, viper.GetString(flags.P2P_Rendezvous), d)
	return d
}

func (d *discovery) OnStart() error {
	if err := d.s.Start(); err != nil {
		d.Logger.Error("failed to start mdns discovery", "err", err)
		return err
	}

	go d.setupDiscover()
	return nil
}

func (d *discovery) OnStop() {
	if err := d.s.Close(); err != nil {
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
	d.Logger.Debug("discovered new peer", "peer", pi)
	err := d.h.Connect(context.Background(), pi)
	if err != nil {
		d.Logger.Debug("error connecting to peer", "peer", pi, "err", err)
		return
	}
	d.Logger.Debug("connected to peer", "peer", pi)

	//peers := n.h.Peerstore().Peers()
	//fmt.Printf("peer len: %d %s\n", len(peers), peers)
	peerNotifier <- pi
}

func (n *Network) notifyPeerFoundEvent() {
	for {
		pi := <-peerNotifier

		n.peerPool[pi.ID.String()] = pi

		gv := version{Height: n.blockChain.LatestBlock().Header.Height.Uint64(),
			Td: n.blockChain.GetTd(),
			ID: n.host.ID().String()}
		data := jointMessage(cVersion, gv.serialize())

		n.SendMessage(pi, data)
	}
}

func (d *discovery) setupDiscover() {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping network of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, d.h)
	if err != nil {
		d.Logger.Error("failed to new dht", "err", err)
		return
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	d.Logger.Debug("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		d.Logger.Error("failed to bootstrap dht", "err", err)
		return
	}
	var routingDiscovery = routing.NewRoutingDiscovery(kademliaDHT)
	rendezvous := viper.GetString(flags.P2P_Rendezvous)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				d.Logger.Error("failed to find peers", "err", err)
				continue
			}

			for p := range peers {
				if p.ID == d.h.ID() {
					continue
				}
				d.Logger.Debug("found peer", "peer", p)
				if d.h.Network().Connectedness(p.ID) != network.Connected {
					_, err = d.h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						d.Logger.Debug("failed to dial peer", "err", err)
						continue
					}
				}
				peerNotifier <- p
			}
		}
	}
}
