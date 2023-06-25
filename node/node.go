package node

import (
	"github.com/DOIDFoundation/node/consensus"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/mempool"
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
	mempool   *mempool.Mempool
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
		network:   network.NewNetwork(chain, logger),
		mempool:   mempool.NewMempool(chain, logger),
	}
	node.BaseService = *service.NewBaseService(logger.With("module", "node"), "Node", node)

	for _, option := range options {
		option(node)
	}

	RegisterAPI(node)
	doid.RegisterAPI(node.chain)
	node.mempool.RegisterAPI()

	return node, nil
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {
	if err := n.rpc.Start(); err != nil {
		return err
	}
	if err := n.mempool.Start(); err != nil {
		return err
	}
	if err := n.network.Start(); err != nil {
		return err
	}
	return nil
}

// OnStop stops the Node. It implements service.Service.
func (n *Node) OnStop() {
	if n.consensus.IsRunning() {
		n.consensus.Stop()
		defer n.Logger.Info("waiting for consensus to finish stopping")
		defer n.consensus.Wait()
	}
	n.rpc.Stop()
	n.network.Stop()
	n.mempool.Stop()

	n.chain.Close()
}

func (n *Node) Chain() *core.BlockChain {
	return n.chain
}
