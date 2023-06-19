package node

import (
	"github.com/DOIDFoundation/node/consensus"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/network"
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

	chain     *core.BlockChain
	consensus *consensus.Consensus
	network   *network.Network
}

// Option sets a parameter for the node.
type Option func(*Node)

// NewNode returns a new, ready to go, CometBFT Node.
func NewNode(logger log.Logger, options ...Option) (*Node, error) {
	chain, err := core.NewBlockChain(logger)
	if err != nil {
		return nil, err
	}

	node := &Node{
		config: &DefaultConfig,
		rpc:    rpc.NewRPC(logger),

		chain:     chain,
		consensus: consensus.New(chain, logger),
		network:   network.NewNetwork(logger),
	}
	node.BaseService = *service.NewBaseService(logger.With("module", "node"), "Node", node)

	for _, option := range options {
		option(node)
	}

	RegisterAPI(node)
	if err := doid.RegisterAPI(node.chain); err != nil {
		return nil, err
	}

	return node, nil
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {
	if err := n.rpc.Start(); err != nil {
		return err
	}
	if err := n.network.Start(); err != nil {
		return err
	}
	if err := n.consensus.Start(); err != nil {
		return err
	}
	return nil
}

// OnStop stops the Node. It implements service.Service.
func (n *Node) OnStop() {
	n.consensus.Stop()
	n.rpc.Stop()
	n.network.Stop()

	n.chain.Close()
}
