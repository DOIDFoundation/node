package core_test

import (
	"os"
	"testing"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBlockChain(t *testing.T) *core.BlockChain {
	viper.SetDefault(flags.DB_Engine, "memdb")
	chain, err := core.NewBlockChain(log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	assert.NoError(t, err)
	return chain
}

func advanceBlock(t *testing.T, chain *core.BlockChain, txs types.Txs, time uint64) {
	newBlock := buildBlock(t, chain, txs, time)
	assert.NoError(t, chain.ApplyBlock(newBlock))
}

func buildBlock(t *testing.T, chain *core.BlockChain, txs types.Txs, time uint64) *types.Block {
	result, err := chain.Simulate(txs)
	assert.NoError(t, err)
	block := chain.LatestBlock()
	header := types.CopyHeader(block.Header)
	header.ParentHash = header.Hash()
	header.Height.Add(header.Height, common.Big1)
	header.Root = result.StateRoot
	header.ReceiptHash = result.ReceiptRoot
	header.Time = time
	newBlock := types.NewBlockWithHeader(header)
	newBlock.Data = types.Data{Txs: txs}
	return newBlock
}

func TestNewBlockchain(t *testing.T) {
	chain := newBlockChain(t)
	assert.Equal(t, common.Big1, chain.LatestBlock().Header.Height)
	assert.NotZero(t, chain.LatestBlock().Header.Root)
	assert.Zero(t, chain.LatestBlock().Header.ParentHash)
	assert.NotZero(t, chain.LatestBlock().Header.TxHash)
	chain.Close()
}

func TestSimulate(t *testing.T) {
	var txs types.Txs
	chain := newBlockChain(t)
	result, err := chain.Simulate(txs)
	assert.NoError(t, err)
	assert.NotZero(t, result.StateRoot)
	assert.NotZero(t, result.ReceiptRoot)
}

func TestApplyBlock(t *testing.T) {
	var txs types.Txs
	chain := newBlockChain(t)
	result, err := chain.Simulate(txs)
	assert.NoError(t, err)
	block := chain.LatestBlock()
	header := types.CopyHeader(block.Header)
	header.ParentHash = header.Hash()
	header.Height.Add(header.Height, common.Big1)
	header.Root = result.StateRoot
	header.ReceiptHash = result.ReceiptRoot
	newBlock := types.NewBlockWithHeader(header)
	newBlock.Data = types.Data{Txs: txs}
	assert.NoError(t, chain.ApplyBlock(newBlock))
	assert.Equal(t, common.Big2, chain.LatestBlock().Header.Height)
	assert.Equal(t, block.Hash(), newBlock.Header.ParentHash)
	assert.Equal(t, newBlock.Hash(), chain.LatestBlock().Hash())
	assert.NotEqual(t, block.Hash(), chain.LatestBlock().Hash())
}

func TestBlockByHeight(t *testing.T) {
	chain := newBlockChain(t)
	assert.Nil(t, chain.BlockByHeight(0))
	assert.NotNil(t, chain.BlockByHeight(1))
	assert.Nil(t, chain.BlockByHeight(2))
	advanceBlock(t, chain, types.Txs{}, 1)
	assert.NotNil(t, chain.BlockByHeight(2))
}

func TestInsertBlocks(t *testing.T) {
	chain := newBlockChain(t)
	blocks := []*types.Block{}
	for i := 0; i < 4; i++ {
		block := buildBlock(t, chain, types.Txs{}, 1)
		require.NoError(t, chain.ApplyBlock(block))
		blocks = append(blocks, block)
	}
	chain.Close()

	// insert into empty chain
	chain = newBlockChain(t)
	assert.NoError(t, chain.InsertBlocks(blocks))
	chain.Close()

	// insert into non-empty chain
	chain = newBlockChain(t)
	for i := 0; i < 5; i++ {
		advanceBlock(t, chain, types.Txs{}, 2)
	}
	assert.Equal(t, core.ErrUnknownAncestor, chain.InsertBlocks(blocks[1:]))
	assert.NoError(t, chain.InsertBlocks(blocks))
}
