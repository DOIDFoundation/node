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
)

func newBlockChain(t *testing.T) *core.BlockChain {
	viper.SetDefault(flags.DB_Engine, "memdb")
	chain, err := core.NewBlockChain(log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	assert.NoError(t, err)
	return chain
}

func advanceBlock(t *testing.T, chain *core.BlockChain, txs types.Txs) {
	hash, _ := chain.Simulate(txs)
	block := chain.LatestBlock()
	header := types.CopyHeader(block.Header)
	header.ParentHash = header.Hash()
	header.Height.Add(header.Height, common.Big1)
	header.Root = hash
	newBlock := types.NewBlockWithHeader(header)
	newBlock.Data = types.Data{Txs: txs}
	assert.NoError(t, chain.ApplyBlock(newBlock))
}

func TestNewBlockchain(t *testing.T) {
	chain := newBlockChain(t)
	assert.Zero(t, chain.LatestBlock().Header.Height.Cmp(common.Big0))
	assert.NotZero(t, chain.LatestBlock().Header.Root)
	assert.Zero(t, chain.LatestBlock().Header.ParentHash)
	assert.NotZero(t, chain.LatestBlock().Header.TxHash)
}

func TestSimulate(t *testing.T) {
	var txs types.Txs
	chain := newBlockChain(t)
	hash, err := chain.Simulate(txs)
	assert.NoError(t, err)
	assert.NotZero(t, hash)
}

func TestApplyBlock(t *testing.T) {
	var txs types.Txs
	chain := newBlockChain(t)
	hash, _ := chain.Simulate(txs)
	block := chain.LatestBlock()
	header := types.CopyHeader(block.Header)
	header.ParentHash = header.Hash()
	header.Height.Add(header.Height, common.Big1)
	header.Root = hash
	newBlock := types.NewBlockWithHeader(header)
	newBlock.Data = types.Data{Txs: txs}
	assert.NoError(t, chain.ApplyBlock(newBlock))
	assert.Zero(t, chain.LatestBlock().Header.Height.Cmp(common.Big1))
	assert.Equal(t, block.Hash(), newBlock.Header.ParentHash)
	assert.Equal(t, newBlock.Hash(), chain.LatestBlock().Hash())
	assert.NotEqual(t, block.Hash(), chain.LatestBlock().Hash())
}

func TestBlockByHeight(t *testing.T) {
	chain := newBlockChain(t)
	assert.NotNil(t, chain.BlockByHeight(0))
	assert.Nil(t, chain.BlockByHeight(1))
	advanceBlock(t, chain, types.Txs{})
	assert.NotNil(t, chain.BlockByHeight(1))
}
