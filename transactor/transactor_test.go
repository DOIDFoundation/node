package transactor_test

import (
	"testing"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newState(t *testing.T) *iavl.MutableTree {
	config.Init()
	db, err := cosmosdb.NewDB("state", cosmosdb.MemDBBackend, "")
	require.NoError(t, err)
	state, err := iavl.NewMutableTree(db, 128, false)
	require.NoError(t, err)
	require.Zero(t, state.Version())
	return state
}

func newTx(t *testing.T, val tx.TypedTx) types.Tx {
	ret, err := tx.NewTx(val)
	require.NoError(t, err)
	return ret
}

func TestApplyTxs(t *testing.T) {
	state := newState(t)
	result, err := transactor.ApplyTxs(state, types.Txs{})
	require.NoError(t, err)
	require.Zero(t, state.Version())
	assert.NotZero(t, result.StateRoot)
	assert.NotNil(t, result)
	assert.NotZero(t, result.TxRoot)
	assert.Empty(t, result.Rejected)
	assert.Empty(t, result.Receipts)

	hash, version, err := state.SaveVersion()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, version)
	assert.NotZero(t, hash)
	assert.EqualValues(t, 1, state.Version())
	assert.EqualValues(t, hash, result.StateRoot)
}
