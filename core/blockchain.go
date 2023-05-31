package core

import (
	"math/big"
	"sync/atomic"

	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
)

type BlockChain struct {
	logger       log.Logger
	blockStore   *store.BlockStore
	currentBlock atomic.Value // Current head of the block chain
}

func NewBlockChain(blockStore *store.BlockStore, logger log.Logger) (*BlockChain, error) {
	bc := &BlockChain{
		blockStore: blockStore,
		logger:     logger.With("module", "blockchain"),
	}

	var block *types.Block = nil
	headBlockHash := blockStore.ReadHeadBlockHash()
	if headBlockHash != nil {
		block = blockStore.ReadBlock(headBlockHash)
		bc.logger.Info("read head block", "hash", headBlockHash, "block", block)
	}
	if block == nil {
		block = types.NewBlockWithHeader(&types.Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(0),
		})
	}
	bc.currentBlock.Store(block)
	return bc, nil
}

func (bc *BlockChain) SetHead(block *types.Block) {
	bc.logger.Info("head block", "head", block.Header)
	bc.blockStore.WriteBlock(block)
	bc.blockStore.WriteHeadBlockHash(block.Hash())
	bc.currentBlock.Store(block)
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}
