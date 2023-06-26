package transactor

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/cosmos/iavl"
)

type ApplyFunc = func(tree *iavl.MutableTree, encoded *encodedtx.EncodedTx) (types.TxStatus, error)

var appliers = make(map[types.TxType]ApplyFunc)

func registerApplyFunc(t types.TxType, f ApplyFunc) {
	appliers[t] = f
}

type rejectedTx struct {
	Index int    `json:"index"`
	Err   string `json:"error"`
}

type ExecutionResult struct {
	StateRoot   types.Hash     `json:"stateRoot"`
	TxRoot      types.Hash     `json:"txRoot"`
	ReceiptRoot types.Hash     `json:"receiptsRoot"`
	Receipts    types.Receipts `json:"receipts"`
	Rejected    []*rejectedTx  `json:"rejected,omitempty"`
}

func ApplyTxs(tree *iavl.MutableTree, txs types.Txs) (*ExecutionResult, error) {
	var (
		rejectedTxs []*rejectedTx
		includedTxs types.Txs
		receipts    = make(types.Receipts, 0)
	)
	for i, t := range txs {
		encoded, err := encodedtx.FromTx(t)
		if err != nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		}
		applier := appliers[encoded.Type]
		if applier == nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, "unknown transaction type"})
			continue
		}
		code, err := applier(tree, encoded)
		if code == types.TxStatusRejected {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		}
		includedTxs = append(includedTxs, t)
		receipt := &types.Receipt{
			TxHash: t.Hash(),
			Status: code,
			Result: []byte(err.Error()),
		}
		receipts = append(receipts, receipt)
	}
	root, err := tree.WorkingHash()
	if err != nil {
		return nil, err
	}
	execRs := &ExecutionResult{
		StateRoot:   root,
		TxRoot:      includedTxs.Hash(),
		ReceiptRoot: receipts.Hash(),
		Receipts:    receipts,
		Rejected:    rejectedTxs,
	}
	return execRs, nil
}
