package encodedtx

import (
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type EncodedTx struct {
	Type types.TxType
	Data []byte
}

func FromTypedTx(val types.TypedTx) (*EncodedTx, error) {
	data, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}
	tx := &EncodedTx{Type: val.Type(), Data: data}
	return tx, nil
}

func FromBytes(b []byte) (*EncodedTx, error) {
	tx := new(EncodedTx)
	if err := rlp.DecodeBytes(b, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func FromTx(b types.Tx) (*EncodedTx, error) {
	return FromBytes(b)
}

func (tx *EncodedTx) Decode(val types.TypedTx) error {
	if tx.Type != val.Type() {
		return errors.New("type mismatch")
	}
	return rlp.DecodeBytes(tx.Data, val)
}

func (tx *EncodedTx) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(tx)
}

func (tx *EncodedTx) ToTx() (types.Tx, error) {
	return rlp.EncodeToBytes(tx)
}
