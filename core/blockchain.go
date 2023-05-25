package core

import (
	"sync/atomic"

	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
)

type BlockChain struct {
	logger       log.Logger
	currentBlock atomic.Value // Current head of the block chain
}

func NewBlockChain(logger log.Logger) (*BlockChain, error) {
	bc := &BlockChain{logger: logger.With("module", "blockchain")}
	var nilBlock *types.Block
	bc.currentBlock.Store(nilBlock)
	return bc, nil
}

func (bc *BlockChain) SetHead(block *types.Block) {
	bc.logger.Info("head block", "head", block.Header)
	bc.currentBlock.Store(block)
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}
