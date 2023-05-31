package node

import (
	"path/filepath"

	"github.com/DOIDFoundation/node/consensus"
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/store"
	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/cli"
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

	blockStore *store.BlockStore
	chain      *core.BlockChain
	consensus  *consensus.Consensus
}

// Option sets a parameter for the node.
type Option func(*Node)

// NewNode returns a new, ready to go, CometBFT Node.
func NewNode(logger log.Logger, options ...Option) (*Node, error) {
	homeDir := viper.GetString(cli.HomeFlag)
	db, err := cmtdb.NewDB("chaindata", cmtdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	blockStore := store.NewBlockStore(db, logger)

	chain, err := core.NewBlockChain(blockStore, logger)
	if err != nil {
		return nil, err
	}

	node := &Node{
		config: &DefaultConfig,
		rpc:    rpc.NewRPC(logger),

		blockStore: blockStore,
		chain:      chain,
		consensus:  consensus.New(chain, logger),
	}
	node.BaseService = *service.NewBaseService(logger, "Node", node)

	for _, option := range options {
		option(node)
	}

	RegisterAPI(node)
	if err := doid.RegisterAPI(node.chain); err != nil {
		db.Close()
		return nil, err
	}

	return node, nil
}

// OnStart starts the Node. It implements service.Service.
func (n *Node) OnStart() error {
	if err := n.rpc.Start(); err != nil {
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
	n.blockStore.Close()
}
