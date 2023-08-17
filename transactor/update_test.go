package transactor_test

import (
	"testing"

	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var owner = types.HexToAddress("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
var owner2 = types.HexToAddress("f39Fd6e51aad88F6F4ce6aB8827279cffFb92265")
var sig = types.Hash("123")
var doidname = "test"

func TestUpdate(t *testing.T) {
	state := newState(t)
	txs := types.Txs{
		newTx(t, &tx.Register{Owner: owner, DOID: doidname}),
		newTx(t, &tx.Update{Owner: owner2, DOID: doidname, Signature: sig}),
	}

	result, err := transactor.ApplyTxs(state, txs)
	require.NoError(t, err)
	require.Zero(t, state.Version())
	assert.NotZero(t, result.StateRoot)
	assert.NotNil(t, result)
	assert.NotZero(t, result.TxRoot)
	assert.Empty(t, result.Rejected)
	assert.NotEmpty(t, result.Receipts)

	hash, version, err := state.SaveVersion()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, version)
	assert.NotZero(t, hash)
	assert.EqualValues(t, 1, state.Version())
	assert.EqualValues(t, hash, result.StateRoot)

	_owner, _ := state.Get(types.DOIDHash(doidname))
	assert.EqualValues(t, _owner, owner2.Bytes())
	ownerStateBytes, _ := state.Get(types.OwnerHash(owner))
	ownerState := &types.OwnerState{}
	rlp.DecodeBytes(ownerStateBytes, ownerState)
	assert.EqualValues(t, ownerState.Names, [][]byte{})

	ownerStateBytes2, _ := state.Get(types.OwnerHash(owner2))
	ownerState2 := &types.OwnerState{}
	rlp.DecodeBytes(ownerStateBytes2, ownerState2)
	tmp := []byte(doidname)
	assert.EqualValues(t, ownerState2.Names, [][]byte{tmp})

}
