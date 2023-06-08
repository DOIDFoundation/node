package encodedtx_test

import (
	"encoding/hex"
	"testing"

	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

func TestDecodeEmptyBytes(t *testing.T) {
	input := []byte{0x80}
	txe, err := encodedtx.FromBytes(input)
	assert.Error(t, err)
	assert.Nil(t, txe)
}

func TestEmptyTx(t *testing.T) {
	// encode
	txe := &encodedtx.EncodedTx{}
	txb, err := txe.ToBytes()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	should, _ := hex.DecodeString("c28080")
	assert.Equal(t, should, txb, "encoded RLP mismatch")

	// decode
	txe, err = encodedtx.FromBytes(should)
	assert.NoError(t, err)
	assert.NotNil(t, txe)
}

func TestTxRegister(t *testing.T) {
	// encode
	txe, err := encodedtx.FromTypedTx(&tx.Register{DOID: "doid"})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	txb, err := rlp.EncodeToBytes(txe)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	should, _ := hex.DecodeString("ca8088c784646f69648080")
	assert.Equal(t, should, txb, "encoded RLP mismatch")

	// decode
	txe, err = encodedtx.FromBytes(should)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, txe)
	var reg tx.Register
	if assert.NoError(t, txe.Decode(&reg)) {
		assert.Equal(t, "doid", reg.DOID)
	}
}
