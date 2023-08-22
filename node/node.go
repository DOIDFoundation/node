package node

import (
	"errors"

	"github.com/DOIDFoundation/node/consensus"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/network"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/spf13/viper"
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
		config:  &DefaultConfig,
		rpc:     rpc.NewRPC(logger),
		chain:   chain,
		network: network.NewNetwork(chain, logger),
		mempool: mempool.NewMempool(chain, logger),
	}
	if node.rpc == nil {
		return nil, errors.New("failed to setup RPC")
	}
	if node.network == nil {
		return nil, errors.New("failed to setup network")
	}
	if node.mempool == nil {
		return nil, errors.New("failed to setup mempool")
	}
	if viper.GetBool(flags.Mine_Enabled) {
		node.consensus = consensus.New(chain, node.mempool, logger)
		if node.consensus == nil {
			return nil, errors.New("failed to setup consensus")
		}
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
	n.network.Stop()
	n.network.Wait()

	if n.consensus != nil && n.consensus.IsRunning() {
		n.consensus.Stop()
		n.consensus.Wait()
	}

	n.rpc.Stop()
	n.mempool.Stop()

	n.rpc.Wait()
	n.mempool.Wait()

	n.chain.Close()
}

func (n *Node) Chain() *core.BlockChain {
	return n.chain
}
