package node

import (
	"fmt"

	"github.com/DOIDFoundation/node/config"
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
	return NodeStatus{IsRuning: api.node.IsRunning(), NetworkId: fmt.Sprint(config.NetworkID)}
}

func RegisterAPI(node *Node) {
	rpc.RegisterName("node", &API{node: node})
}
