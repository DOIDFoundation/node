package consensus_test

import (
	"os"
	"testing"

	"github.com/DOIDFoundation/node/consensus"
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

func newConsensus(t *testing.T) (*consensus.Consensus, *mempool.Mempool, *core.BlockChain) {
	viper.SetDefault(flags.DB_Engine, "memdb")
	viper.Set(flags.Mine_Miner, "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	chain, err := core.NewBlockChain(logger)
	chain.LatestBlock().Header.Difficulty.SetUint64(1)
	require.NoError(t, err)
	txpool := mempool.NewMempool(chain, logger)
	require.NotNil(t, txpool)
	c := consensus.New(chain, txpool, logger)
	require.NotNil(t, c)
	return c, txpool, chain
}

func TestWork(t *testing.T) {
	c, _, _ := newConsensus(t)
	c.Start()
	var block *types.Block
	events.NewMinedBlock.Subscribe("test", func(data *types.Block) {
		block = data
		c.Stop()
	})
	c.Wait()
	assert.NotNil(t, block)
	assert.Positive(t, block.Header.Difficulty.Cmp(common.Big1))
}

func TestWorkWithTx(t *testing.T) {
	c, txpool, chain := newConsensus(t)
	genesis := chain.LatestBlock()
	reg, err := tx.NewTx(&tx.Register{DOID: "test"})
	require.NoError(t, err)
	require.NoError(t, txpool.AddLocal(reg))
	c.Start()
	var block *types.Block
	events.NewMinedBlock.Subscribe("test", func(data *types.Block) {
		block = data
		c.Stop()
	})
	c.Wait()
	assert.NotNil(t, block)
	assert.NotEmpty(t, block.Txs)
	assert.NotEqual(t, block.Header.Root, genesis.Header.Root)
}
