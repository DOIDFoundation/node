package transactor_test

import (
	"testing"

	"github.com/DOIDFoundation/node/transactor"
	"github.com/cometbft/cometbft/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTxs(t *testing.T) {
	db, err := cosmosdb.NewDB("state", cosmosdb.MemDBBackend, "")
	require.NoError(t, err)
	state, err := iavl.NewMutableTree(db, 128, false)
	require.NoError(t, err)
	require.Zero(t, state.Version())
	result, err := transactor.ApplyTxs(state, types.Txs{})
	require.NoError(t, err)
	require.Zero(t, state.Version())
	assert.NotZero(t, result.StateRoot)
	assert.NotNil(t, result)
	hash, version, err := state.SaveVersion()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, version)
	assert.NotZero(t, hash)
	assert.EqualValues(t, 1, state.Version())
	assert.EqualValues(t, hash, result.StateRoot)
}
