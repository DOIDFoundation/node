package core_test

import (
	"testing"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendBlocks(t *testing.T) {
	chain := newBlockChain(t)
	blocks := []*types.Block{}
	for i := 1; i < 10; i++ {
		block := buildBlock(t, chain, types.Txs{}, uint64(i))
		assert.NoError(t, chain.ApplyBlock(block))
		blocks = append(blocks, block)
	}

	// insert all known blocks
	hc := chain.NewHeaderChain()
	assert.NoError(t, hc.AppendBlocks(blocks[:5]))
	assert.NoError(t, hc.AppendBlocks(blocks[:3]))
	assert.NoError(t, hc.AppendBlocks(blocks[1:6]))
	assert.NoError(t, hc.AppendBlocks(blocks[4:]))
	assert.NoError(t, hc.AppendBlocks(blocks[6:]))
	assert.NoError(t, hc.AppendBlocks(blocks[5:]))
	assert.Nil(t, hc.GetTd())

	chain.Close()

	// insert into non-empty chain
	chain = newBlockChain(t)
	for i := 2; i < 10; i++ {
		advanceBlock(t, chain, types.Txs{}, uint64(i))
	}
	hc = chain.NewHeaderChain()
	assert.NoError(t, hc.AppendBlocks(blocks))
	assert.Negative(t, chain.GetTd().Cmp(hc.GetTd()))
	advanceBlock(t, chain, types.Txs{}, 100)
	assert.Negative(t, chain.GetTd().Cmp(hc.GetTd()))
	advanceBlock(t, chain, types.Txs{}, 101)
	assert.Positive(t, chain.GetTd().Cmp(hc.GetTd()))

	hc = chain.NewHeaderChain()
	assert.EqualError(t, hc.AppendBlocks(blocks[3:]), core.ErrUnknownAncestor.Error())
	blocks[1].Header.Time = 1
	assert.EqualError(t, hc.AppendBlocks(blocks), types.ErrNotContiguous.Error())
}

func TestCanStartFrom(t *testing.T) {
	chain := newBlockChain(t)
	blocks := []*types.Block{}
	for i := 1; i < 10; i++ {
		block := buildBlock(t, chain, types.Txs{}, uint64(i))
		assert.NoError(t, chain.ApplyBlock(block), block.Header)
		blocks = append(blocks, block)
	}

	// can start from known blocks
	hc := chain.NewHeaderChain()
	for _, block := range blocks {
		assert.True(t, hc.CanStartFrom(uint64(block.Header.Height.Int64()), block.Header.Hash()), block.Header)
	}

	// can not start from unknown blocks
	chain = newBlockChain(t)
	for _, block := range blocks[:5] {
		require.NoError(t, chain.ApplyBlock(block), block.Header)
	}
	hc = chain.NewHeaderChain()
	for _, block := range blocks[5:] {
		assert.False(t, hc.CanStartFrom(uint64(block.Header.Height.Int64()), block.Header.Hash()), block.Header)
	}

	// can start from cached blocks
	require.NoError(t, hc.AppendBlocks(blocks[5:7]))
	hc = chain.NewHeaderChain()
	for _, block := range blocks[5:7] {
		assert.True(t, hc.CanStartFrom(uint64(block.Header.Height.Int64()), block.Header.Hash()), block.Header)
	}
	// can not start from unknown blocks
	hc = chain.NewHeaderChain()
	for _, block := range blocks[7:] {
		assert.False(t, hc.CanStartFrom(uint64(block.Header.Height.Int64()), block.Header.Hash()), block.Header)
	}

	// remove a cached block
	chain.BlockStore().DeleteBlock(blocks[5].Header.Height.Uint64(), blocks[5].Hash())
	// can not start from cached and unknown blocks
	for _, block := range blocks[5:] {
		assert.False(t, hc.CanStartFrom(uint64(block.Header.Height.Int64()), block.Header.Hash()), block.Header)
	}
}
