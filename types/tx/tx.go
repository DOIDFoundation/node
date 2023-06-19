package tx

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
)

func NewTx(val types.TypedTx) (types.Tx, error) {
	encoded, err := encodedtx.FromTypedTx(val)
	if err != nil {
		return nil, err
	}
	return encoded.ToTx()
}

func Decode(tx types.Tx, val types.TypedTx) error {
	encoded, err := encodedtx.FromTx(tx)
	if err != nil {
		return err
	}
	err = encoded.Decode(val)
	if err != nil {
		return err
	}
	return nil
}
