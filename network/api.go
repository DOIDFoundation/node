package network

import (
	"github.com/DOIDFoundation/node/rpc"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type API struct {
	net *Network
}

type Status struct {
	PeersInTopic peer.IDSlice `json:"peersInTopic"`
	Connections  []string     `json:"connections"`
}

func (api *API) Status() Status {
	return Status{
		PeersInTopic: api.net.discovery.topic.ListPeers(),
		Connections:  api.Connections(),
	}
}

func (api *API) PeersInStore() peer.IDSlice {
	return api.net.host.Peerstore().Peers()
}

func (api *API) Peers() peer.IDSlice {
	return api.net.host.Network().Peers()
}

func (api *API) Connections() (addrs []string) {
	for _, conn := range api.net.host.Network().Conns() {
		a := peer.AddrInfo{ID: conn.RemotePeer(), Addrs: []multiaddr.Multiaddr{conn.RemoteMultiaddr()}}
		addrs = append(addrs, a.String())
	}
	return
}

func RegisterAPI(net *Network) {
	api := &API{net: net}
	rpc.RegisterName("network", api)
}
