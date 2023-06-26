package types

import (
	"github.com/cometbft/cometbft/crypto/merkle"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type (
	Address    = cmttypes.Address
	BlockNonce = ethtypes.BlockNonce
	Data       = cmttypes.Data
	HexBytes   = cmtbytes.HexBytes
	Tx         = cmttypes.Tx
	TxHash     = cmttypes.TxKey
	Txs        = cmttypes.Txs
	TxType     = uint8
)

var EncodeNonce = ethtypes.EncodeNonce

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
