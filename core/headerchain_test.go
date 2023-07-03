package core_test

import (
	"testing"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/types"
	"github.com/stretchr/testify/assert"
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
