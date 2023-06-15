package transactor

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/cosmos/iavl"
)

func ApplyTxs(tree *iavl.MutableTree, txs types.Txs) (types.Hash, error) {
	for _, t := range txs {
		encoded, err := encodedtx.FromTx(t)
		if err != nil {
			return nil, err
		}
		switch encoded.Type {
		case types.TxTypeRegister:
			applyRegister(tree, encoded)
		}
	}
	return tree.WorkingHash()
}
