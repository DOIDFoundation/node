package node

import (
	"github.com/DOIDFoundation/node/rpc"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
)

//------------------------------------------------------------------------------

// Node is the highest level interface to a full node.
// It includes all configuration information and running services.
type Node struct {
	service.BaseService
	config *Config
	rpc    *rpc.RPC
}

// Option sets a parameter for the node.
type Option func(*Node)

// NewNode returns a new, ready to go, CometBFT Node.
func NewNode(logger log.Logger, options ...Option) (*Node, error) {
	node := &Node{
		config: &DefaultConfig,
		rpc:    rpc.NewRPC(logger),
	}
	node.BaseService = *service.NewBaseService(logger, "Node", node)

	for _, option := range options {
		option(node)
	}

	RegisterAPI(node)

	return node, nil
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {
	if err := n.rpc.Start(); err != nil {
		return err
	}
	return nil
}

// OnStop stops the Node. It implements service.Service.
func (n *Node) OnStop() {
	n.rpc.Stop()
}
