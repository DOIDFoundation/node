package transactor

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

type Transactor interface {
	Validate(tree *iavl.ImmutableTree, t tx.TypedTx) error
	Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error)
}

type resultCode = uint8

// Transaction result codes, append only.
const (
	resSuccess  resultCode = iota
	resFailed              // will be included in block
	resRejected            // should be rejected from block
	resLater               // should be retried later
)

var transactors = map[tx.Type]Transactor{}

func registerTransactor(t tx.Type, f Transactor) {
	transactors[t] = f
}

type rejectedTx struct {
	Index int    `json:"index"`
	Err   string `json:"error"`
}

type ExecutionResult struct {
	StateRoot   types.Hash     `json:"stateRoot"`
	TxRoot      types.Hash     `json:"txRoot"`
	Txs         types.Txs      `json:"txs"`
	ReceiptRoot types.Hash     `json:"receiptsRoot"`
	Receipts    types.Receipts `json:"receipts"`
	Rejected    []*rejectedTx  `json:"rejected,omitempty"`
	Pending     []int          `json:"pending,omitempty"`
}

func ValidateTx(tree *iavl.ImmutableTree, t tx.TypedTx) error {
	transactor := transactors[t.Type()]
	return transactor.Validate(tree, t)
}

func ApplyTxs(tree *iavl.MutableTree, txs types.Txs) (*ExecutionResult, error) {
	var (
		rejectedTxs []*rejectedTx
		pendingTxs  []int
		includedTxs types.Txs
		receipts    = make(types.Receipts, 0)
	)
	for i, t := range txs {
		decoded, err := tx.Decode(t)
		if err != nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		}
		transactor := transactors[decoded.Type()]
		if transactor == nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, "unknown transaction type"})
			continue
		}
		code, err := transactor.Apply(tree, decoded)
		switch code {
		case resRejected:
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		case resLater:
			pendingTxs = append(pendingTxs, i)
			continue
		}
		includedTxs = append(includedTxs, t)
		receipt := &types.Receipt{
			TxHash: t.Hash(),
		}
		if err != nil {
			receipt.Result = []byte(err.Error())
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
		Txs:         includedTxs,
		ReceiptRoot: receipts.Hash(),
		Receipts:    receipts,
		Rejected:    rejectedTxs,
		Pending:     pendingTxs,
	}
	return execRs, nil
}
