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

func NewDiscovery(h host.Host, logger log.Logger) *discovery {
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
func (n *discovery) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID.String() == n.h.ID().String() {
		return
	}
	n.Logger.Debug("discovered new peer", "peer", pi)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		n.Logger.Debug("error connecting to peer", "peer", pi, "err", err)
		return
	}
	n.Logger.Debug("connected to peer", "peer", pi)

	//peers := n.h.Peerstore().Peers()
	//fmt.Printf("peer len: %d %s\n", len(peers), peers)
	peerNotifier <- pi
}

func (n *Network) notifyPeerFoundEvent() {
	for {
		pi := <-peerNotifier

		n.peerPool[pi.ID.String()] = pi

		gv := version{Height: n.blockChain.LatestBlock().Header.Height.Uint64(), ID: n.host.ID().String()}
		data := n.jointMessage(cVersion, gv.serialize())

		n.SendMessage(pi, data)
	}
}

func setupDiscover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	var routingDiscovery = routing.NewRoutingDiscovery(dht)
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
				panic(err)
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						continue
					}
				}
			}
		}
	}
}
