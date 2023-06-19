package transactor

import (
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

func applyRegister(tree *iavl.MutableTree, encoded *encodedtx.EncodedTx) error {
	var register tx.Register
	if err := encoded.Decode(&register); err != nil {
		return err
	}
	_, err := tree.Set([]byte(register.DOID), register.Owner)
	return err
}
