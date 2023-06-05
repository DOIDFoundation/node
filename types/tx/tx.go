package tx

import (
	"errors"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type Tx interface {
	Type() Type
}

type EncodedTx struct {
	Type Type
	Data []byte
}

func Encode(val Tx) (*EncodedTx, error) {
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

func (tx *EncodedTx) Decode(val Tx) error {
	if tx.Type != val.Type() {
		return errors.New("type mismatch")
	}
	return rlp.DecodeBytes(tx.Data, val)
}

func (tx *EncodedTx) ToBytes() ([]byte, error) {
	return rlp.EncodeToBytes(tx)
}

func DecodeTx(b cmttypes.Tx, val Tx) error {
	tx, err := FromBytes(b)
	if err != nil {
		return err
	}
	err = tx.Decode(val)
	if err != nil {
		return err
	}
	return nil
}
