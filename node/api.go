package node

import (
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/rpc"
)

type API struct {
	node *Node
}

type NodeStatus struct {
	IsRuning  bool   `json:"is_runing"`
	NetworkId string `json:"network_id"`
}

func (api *API) Status() NodeStatus {
	return NodeStatus{IsRuning: api.node.IsRunning(), NetworkId: flags.NetworkId}
}

func RegisterAPI(node *Node) {
	rpc.RegisterName("node", &API{node: node})
}
