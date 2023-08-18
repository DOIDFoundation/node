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
	Connections  Connections  `json:"connections"`
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

type Connections struct {
	Num         int                      `json:"num"`
	Connections []map[string]interface{} `json:"connections"`
}

func (api *API) Connections() (c Connections) {
	c.Num = len(api.net.host.Network().Conns())
	for _, conn := range api.net.host.Network().Conns() {
		a := peer.AddrInfo{ID: conn.RemotePeer(), Addrs: []multiaddr.Multiaddr{conn.RemoteMultiaddr()}}
		ret := a.Loggable()
		if agentVersion, err := api.net.host.Peerstore().Get(a.ID, "AgentVersion"); err == nil {
			ret["version"] = agentVersion
		}
		c.Connections = append(c.Connections, ret)
	}
	return
}

func RegisterAPI(net *Network) {
	api := &API{net: net}
	rpc.RegisterName("network", api)
}
