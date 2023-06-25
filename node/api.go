package node

import (
	"github.com/DOIDFoundation/node/rpc"
)

type API struct {
	node *Node
}

func (api *API) Status() map[string]bool {
	return map[string]bool{"is_running": api.node.IsRunning()}
}

func RegisterAPI(node *Node) {
	rpc.RegisterName("node", &API{node: node})
}
