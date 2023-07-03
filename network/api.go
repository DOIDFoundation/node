package network

import (
	"github.com/DOIDFoundation/node/rpc"
	"github.com/libp2p/go-libp2p/core/peer"
)

type API struct {
	net *Network
}

type Status struct {
	IsRunning bool `json:"is_running"`
}

func (api *API) Status() Status {
	return Status{IsRunning: api.net.IsRunning()}
}

func (api *API) PeersInStore() peer.IDSlice {
	return api.net.host.Peerstore().Peers()
}

func (api *API) Peers() peer.IDSlice {
	return api.net.host.Network().Peers()
}

func RegisterAPI(net *Network) {
	api := &API{net: net}
	rpc.RegisterName("network", api)
}
