package transactor

import (
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

func applyRegister(tree *iavl.MutableTree, encoded *encodedtx.EncodedTx) (resultCode, error) {
	var register tx.Register
	if err := encoded.Decode(&register); err != nil {
		return resRejected, err
	}
	key := []byte(register.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return resRejected, err
	}

	if has {
		return resRejected, errors.New("name exists")
	}

	_, err = tree.Set(key, register.Owner)
	if err != nil {
		return resRejected, err
	}
	return resSuccess, nil
}

func init() {
	registerApplyFunc(types.TxTypeRegister, applyRegister)
}
