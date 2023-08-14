package tx

import (
	"errors"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/viper"
)

// TypedTx is the interface of all kinds of transactions
type TypedTx interface {
	Type() Type // return type of transaction
}

// field type overrides for gencodec
type txMarshaling struct {
	Type Type `json:"type"` // adds call to Type() in MarshalJSON
}

func Version(t types.Tx) uint8 {
	return t[0]
}

func ChainId(t types.Tx) uint8 {
	return t[1]
}

func TxType(t types.Tx) Type {
	return Type(t[2])
}

func TxData(t types.Tx) []byte {
	return t[3:]
}

// Get a raw Tx from val.
func NewTx(val TypedTx) (types.Tx, error) {
	data, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, len(data)+3)
	ret[0] = 0
	ret[1] = byte(viper.GetInt(flags.NetworkId))
	ret[2] = byte(val.Type())
	copy(ret[3:], data)
	return ret, nil
}

func Decode(b types.Tx) (TypedTx, error) {
	switch Version(b) {
	case 0:
		ret := NewTypedTx(TxType(b))
		if err := rlp.DecodeBytes(TxData(b), ret); err != nil {
			return nil, err
		}
		return ret, nil
	}
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

// encodedTx holds tx type in Type and rlp encoded TypedTx in Data. deprecated.
type encodedTx struct {
	Type Type   // type of encoded TypedTx
	Data []byte // rlp encoded TypedTx
}
