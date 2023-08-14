package node

import (
	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/rpc"
)

type API struct {
	node *Node
}

type NodeStatus struct {
	IsRuning  bool `json:"is_runing"`
	NetworkId byte `json:"network_id"`
}

func (api *API) Status() NodeStatus {
	return NodeStatus{IsRuning: api.node.IsRunning(), NetworkId: config.NetworkID}
}

func RegisterAPI(node *Node) {
	rpc.RegisterName("node", &API{node: node})
}
