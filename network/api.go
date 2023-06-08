package network

import "github.com/DOIDFoundation/node/rpc"

type API struct {
	node *Node
}

type Status struct {
	IsRunning bool `json:"is_running"`
}

func (api *API) Status() Status {
	return Status{IsRunning: api.node.IsRunning()}
}

func RegisterAPI(node *Node) {
	api := &API{node: node}
	rpc.RegisterName("network", api)
}
