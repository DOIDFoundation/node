package store_test

import (
	"testing"

	"github.com/DOIDFoundation/node/types"
	"github.com/stretchr/testify/assert"
)

func TestHashByHeight(t *testing.T) {
	store := newBlockStore(t)
	for i := uint64(0); i < 10; i++ {
		store.WriteHashByHeight(i, types.Hash{byte(i)}, types.HexToAddress("0x0000"))
	}
	for i := uint64(0); i < 10; i++ {
		assert.Equal(t, types.Hash{byte(i)}, store.ReadHashByHeight(i))
	}
	store.DeleteHashByHeight(5)
	assert.NotPanics(t, func() {
		store.DeleteHashByHeight(5)
	})
	assert.Nil(t, store.ReadHashByHeight(5))
	assert.NotNil(t, store.ReadHashByHeight(6))
	store.DeleteHashByHeightFrom(5)
	for i := uint64(0); i < 5; i++ {
		assert.NotNil(t, store.ReadHashByHeight(i))
	}
	for i := uint64(6); i < 10; i++ {
		assert.Nil(t, store.ReadHashByHeight(i))
	}
	assert.NotPanics(t, func() {
		store.DeleteHashByHeightFrom(11)
	})
}

func TestHeadHash(t *testing.T) {
	store := newBlockStore(t)
	store.WriteHeadBlockHash(types.Hash{1})
	assert.Equal(t, types.Hash{byte(1)}, store.ReadHeadBlockHash())
	assert.Nil(t, store.ReadHeadBlock())
	block := types.NewBlockWithHeader(&types.Header{})
	store.WriteBlock(block)
	store.WriteHeadBlockHash(block.Hash())
	assert.Equal(t, block.Hash(), store.ReadHeadBlockHash())
	assert.NotNil(t, store.ReadHeadBlock())
}
