package types

import (
	"math/big"

	"github.com/cometbft/cometbft/crypto/merkle"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type (
	Address    = cmttypes.Address
	BlockNonce = ethtypes.BlockNonce
	HexBytes   = cmtbytes.HexBytes
	Tx         = cmttypes.Tx
	TxHash     = cmttypes.TxKey
	Txs        = cmttypes.Txs
	TxType     = uint8
)

func HexToAddress(s string) Address {
	return common.FromHex(s)
}

var EncodeNonce = ethtypes.EncodeNonce

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Txs) Txs {
	keep := make(Txs, 0, len(a))

	remove := make(map[TxHash]struct{})
	for _, tx := range b {
		remove[tx.Key()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Key()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// Transaction types, append only.
const (
	TxTypeRegister TxType = iota
)

type TypedTx interface {
	Type() TxType
}

type Receipt struct {
	TxHash Hash     `json:"transactionHash"`
	Result HexBytes `json:"result"`
	Logs   HexBytes `json:"logs"`
}

type StoredReceipt struct {
	Receipt
	// Inclusion information: These fields provide information about the inclusion of the
	// transaction corresponding to this receipt.
	BlockHash        Hash     `json:"blockHash,omitempty"`
	BlockNumber      *big.Int `json:"blockNumber,omitempty"`
	TransactionIndex uint     `json:"transactionIndex"`
}

func (r *Receipt) Hash() Hash {
	return rlpHash(r)
}

func (r *Receipt) Success() bool {
	return len(r.Result) == 0
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
