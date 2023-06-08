package tx

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
)

func Decode(tx types.Tx, val types.TypedTx) error {
	encoded, err := encodedtx.FromBytes(tx)
	if err != nil {
		return err
	}
	err = encoded.Decode(val)
	if err != nil {
		return err
	}
	return nil
}
