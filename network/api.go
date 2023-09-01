package network

import (
	"context"

	"github.com/DOIDFoundation/node/rpc"
	"github.com/libp2p/go-libp2p/core/peer"
)

type API struct {
	net *Network
}

type PrivateAPI struct {
	net *Network
}

func (api *API) Status() map[string]int {
	return map[string]int{
		"peers":        len(api.net.host.Network().Peers()),
		"peersInTopic": len(api.net.discovery.topic.ListPeers()),
		"connections":  len(api.net.host.Network().Conns()),
	}
}

func (api *API) Peers() peer.IDSlice {
	return api.net.host.Network().Peers()
}

func (api *API) PeersInStore() peer.IDSlice {
	return api.net.host.Peerstore().Peers()
}

func (api *API) PeersInTopic() peer.IDSlice {
	return api.net.discovery.topic.ListPeers()
}

func (api *API) PeersWithVersion() (c []map[string]interface{}) {
	for _, p := range api.net.host.Network().Peers() {
		ret := map[string]interface{}{
			"peerID": p,
			"addrs":  api.net.host.Peerstore().Addrs(p),
			"conns":  len(api.net.host.Network().ConnsToPeer(p)),
		}
		if agentVersion, err := api.net.host.Peerstore().Get(p, "AgentVersion"); err == nil {
			ret["version"] = agentVersion
		}
		if _, ok := peerHasState.Load(p); ok {
			peerState := getPeerState(api.net.host.Peerstore(), p)
			ret["height"] = peerState.Height
			ret["td"] = peerState.Td
		}
		c = append(c, ret)
	}
	return
}

type Connections struct {
	Num         int                      `json:"num"`
	Connections []map[string]interface{} `json:"connections"`
}

func (api *API) Connections() (c Connections) {
	c.Num = len(api.net.host.Network().Conns())
	for _, conn := range api.net.host.Network().Conns() {
		id := conn.RemotePeer()
		ret := map[string]interface{}{
			"peer": id,
			"addr": conn.RemoteMultiaddr(),
		}
		if agentVersion, err := api.net.host.Peerstore().Get(id, "AgentVersion"); err == nil {
			ret["version"] = agentVersion
		}
		if _, ok := peerHasState.Load(id); ok {
			peerState := getPeerState(api.net.host.Peerstore(), id)
			ret["height"] = peerState.Height
			ret["td"] = peerState.Td
		}
		c.Connections = append(c.Connections, ret)
	}
	return
}

func (api *PrivateAPI) Connect(ctx context.Context, s string) error {
	addr, err := peer.AddrInfoFromString(s)
	if err != nil {
		return err
	}
	if err = api.net.host.Connect(ctx, *addr); err != nil {
		return err
	}
	return nil
}

func RegisterAPI(net *Network) {
	rpc.RegisterName("network", &API{net: net})
	rpc.RegisterName("admin", &PrivateAPI{net: net})
}
