package tx

import (
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// TypedTx is the interface of all kinds of transactions
type TypedTx interface {
	Type() Type // return type of transaction
}

// field type overrides for gencodec
type txMarshaling struct {
	Type Type `json:"type"` // adds call to Type() in MarshalJSON
}

// encodedTx holds tx type in Type and rlp encoded TypedTx in Data.
type encodedTx struct {
	Type Type   // type of encoded TypedTx
	Data []byte // rlp encoded TypedTx
}

// Get a raw Tx from val.
func NewTx(val TypedTx) (types.Tx, error) {
	data, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(&encodedTx{Type: val.Type(), Data: data})
}

func Decode(b types.Tx) (TypedTx, error) {
	var enc encodedTx
	if err := rlp.DecodeBytes(b, &enc); err != nil {
		return nil, err
	}
	ret := NewTypedTx(enc.Type)

	if enc.Type != ret.Type() {
		return nil, errors.New("type mismatch")
	}
	if err := rlp.DecodeBytes(enc.Data, ret); err != nil {
		return nil, err
	}
	return ret, nil
}
