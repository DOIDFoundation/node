package transactor_test

import (
	"testing"

	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	state := newState(t)
	txs := types.Txs{
		newTx(t, &tx.Register{DOID: "test"}),
		newTx(t, &tx.Register{DOID: "test"}),
	}

	result, err := transactor.ApplyTxs(state, txs)
	require.NoError(t, err)
	require.Zero(t, state.Version())
	assert.NotZero(t, result.StateRoot)
	assert.NotNil(t, result)
	assert.NotZero(t, result.TxRoot)
	assert.NotEmpty(t, result.Rejected)
	assert.NotEmpty(t, result.Receipts)

	hash, version, err := state.SaveVersion()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, version)
	assert.NotZero(t, hash)
	assert.EqualValues(t, 1, state.Version())
	assert.EqualValues(t, hash, result.StateRoot)
}
