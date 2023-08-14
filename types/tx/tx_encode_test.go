package tx_test

import (
	"encoding/hex"
	"testing"

	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestDecodeEmptyBytes(t *testing.T) {
	input := []byte{0x80}
	txe, err := tx.Decode(input)
	assert.Error(t, err)
	assert.Nil(t, txe)
}

func TestEmptyTx(t *testing.T) {
	should, _ := hex.DecodeString("c28080")
	// decode
	txp, err := tx.Decode(should)
	assert.Error(t, err)
	assert.Nil(t, txp)
}

func TestTxRegister(t *testing.T) {
	should, _ := hex.DecodeString("000000c784646f69648080")
	// encode
	txe, err := tx.NewTx(&tx.Register{DOID: "doid"})
	assert.NoError(t, err)
	assert.EqualValues(t, should, txe, "encoded RLP mismatch")

	// decode
	txp, err := tx.Decode(should)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, txp)
	assert.NotNil(t, txp.(*tx.Register))

	// decode old version
	txp, err = tx.Decode(hexutil.MustDecode("0xca8088c784646f69648080"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, txp)
	assert.NotNil(t, txp.(*tx.Register))
}
