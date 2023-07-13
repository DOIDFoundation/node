package mempool_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMempool(t *testing.T) (*mempool.Mempool, *core.BlockChain) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	viper.SetDefault(flags.DB_Engine, "memdb")
	chain, err := core.NewBlockChain(logger)
	require.NoError(t, err)
	p := mempool.NewMempool(chain, logger)
	require.NotNil(t, p)
	return p, chain
}

func buildBlock(t *testing.T, chain *core.BlockChain, txs types.Txs, time uint64) *types.Block {
	result, err := chain.Simulate(txs)
	assert.NoError(t, err)
	parent := chain.LatestBlock()
	header := types.CopyHeader(parent.Header)
	header.ParentHash = parent.Hash()
	header.Height.Add(header.Height, common.Big1)
	header.Root = result.StateRoot
	header.ReceiptHash = result.ReceiptRoot
	header.TxHash = result.TxRoot
	header.Time = time
	header.Difficulty = types.CalcDifficulty(time, parent.Header)
	newBlock := types.NewBlockWithHeader(header)
	newBlock.Txs = result.Txs
	newBlock.Receipts = result.Receipts
	return newBlock
}

func TestMempool(t *testing.T) {
	p, c := newMempool(t)
	txe, err := tx.NewTx(&tx.Register{DOID: "doid"})
	require.NoError(t, err)
	assert.NoError(t, p.AddLocal(txe))
	assert.ErrorIs(t, p.AddLocal(txe), mempool.ErrAlreadyKnown)
	var wg sync.WaitGroup
	wg.Add(1)
	c.ApplyBlock(buildBlock(t, c, p.Pending(), 1))
	c.ApplyBlock(buildBlock(t, c, p.Pending(), 2))
	p.Start()
	t.Cleanup(func() { p.Stop(); p.Wait() })
	events.NewChainHead.Subscribe("test", func(data *types.Block) {
		for i := 0; i < 10 && len(p.Pending()) > 0; i++ {
			time.Sleep(100 * time.Millisecond)
		}
		assert.Empty(t, p.Pending())
		wg.Done()
	})
	c.ApplyBlock(buildBlock(t, c, p.Pending(), 3))
	wg.Wait()
}
