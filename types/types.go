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

// TxDifference returns two sets which are dropped and kept from a compared to b.
func TxDifference(a, b Txs) (dropped, kept Txs) {
	dropped = make(Txs, 0, len(a))
	kept = make(Txs, 0, len(a))

	check := make(map[TxHash]Tx)
	for _, tx := range b {
		check[tx.Key()] = tx
	}

	for _, tx := range a {
		if _, ok := check[tx.Key()]; !ok {
			dropped = append(dropped, tx)
		} else {
			kept = append(kept, tx)
			delete(check, tx.Key())
		}
	}
	for _, v := range check {
		kept = append(kept, v)
	}

	return dropped, kept
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
