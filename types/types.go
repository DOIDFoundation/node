package types

import (
	"github.com/cometbft/cometbft/crypto/merkle"
	cmttypes "github.com/cometbft/cometbft/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type Address = cmttypes.Address
type BlockNonce = ethtypes.BlockNonce
type Data = cmttypes.Data
type Tx = cmttypes.Tx
type TxHash = cmttypes.TxKey
type Txs = cmttypes.Txs

type TxType = uint8

// Transaction types, append only.
const (
	TxTypeRegister TxType = iota
)

type TypedTx interface {
	Type() TxType
}

type TxStatus = uint8

// Transaction types, append only.
const (
	TxStatusSuccess  TxStatus = iota
	TxStatusRejected          // Internal error, should not be exposed to users
	TxStatusFailed
)

var EncodeNonce = ethtypes.EncodeNonce

type Receipt struct {
	TxHash Hash     `json:"transactionHash"`
	Status TxStatus `json:"status"`
	Result []byte   `json:"logs"`
}

func (r *Receipt) Hash() Hash {
	return rlpHash(r)
}

type Receipts []*Receipt

func (rs Receipts) Hash() []byte {
	hl := rs.hashList()
	return merkle.HashFromByteSlices(hl)
}

func (rs Receipts) hashList() [][]byte {
	hl := make([][]byte, len(rs))
	for i := 0; i < len(rs); i++ {
		hl[i] = rs[i].Hash()
	}
	return hl
}
