package transactor

import (
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

func applyRegister(tree *iavl.MutableTree, encoded *encodedtx.EncodedTx) (types.TxStatus, error) {
	var register tx.Register
	if err := encoded.Decode(&register); err != nil {
		return types.TxStatusRejected, err
	}
	key := []byte(register.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return types.TxStatusRejected, err
	}

	if has {
		return types.TxStatusFailed, errors.New("name exists")
	}

	_, err = tree.Set(key, register.Owner)
	if err != nil {
		return types.TxStatusRejected, err
	}
	return types.TxStatusSuccess, nil
}

func init() {
	registerApplyFunc(types.TxTypeRegister, applyRegister)
}
