package types

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/cometbft/cometbft/crypto/merkle"
)

type Block struct {
	Header   *Header `json:"header"`
	Txs      `json:"txs"`
	Uncles   Headers `json:"uncles"`
	Receipts `json:"receipts"`

	// // These fields are used by package eth to track
	// // inter-peer block relay.
	// ReceivedAt   time.Time
	// ReceivedFrom interface{}
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{Header: CopyHeader(header)}
}

func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Height = new(big.Int); h.Height != nil {
		cpy.Height.Set(h.Height)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
}

// fillHeader fills in any remaining header fields that are a function of the block data
func (b *Block) fillHeader() {
	if b.Header.UncleHash == nil {
		b.Header.UncleHash = b.Uncles.Hash()
	}
	if b.Header.TxHash == nil {
		b.Header.TxHash = b.Txs.Hash()
	}
	if b.Header.ReceiptHash == nil {
		b.Header.ReceiptHash = b.Receipts.Hash()
	}
}

// Hash computes and returns the block hash.
// If the block is incomplete, block hash is nil for safety.
func (b *Block) Hash() Hash {
	if b == nil {
		return nil
	}

	b.fillHeader()
	return b.Header.Hash()
}

type Header struct {
	ParentHash  Hash       `json:"parentHash"       gencodec:"required"`
	UncleHash   Hash       `json:"sha3Uncles"       gencodec:"required"`
	Miner       Address    `json:"miner"            gencodec:"required"`
	Root        Hash       `json:"stateRoot"        gencodec:"required"`
	TxHash      Hash       `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash Hash       `json:"receiptsRoot"     gencodec:"required"`
	Difficulty  *big.Int   `json:"difficulty"       gencodec:"required"`
	Height      *big.Int   `json:"height"           gencodec:"required"`
	Time        uint64     `json:"timestamp"        gencodec:"required"`
	Extra       HexBytes   `json:"extraData"        gencodec:"required"`
	Nonce       BlockNonce `json:"nonce"`
}

func (h *Header) Hash() Hash {
	return rlpHash(h)
}

func (h *Header) IsValid(parent *Header) error {
	if h.Height.Uint64() != parent.Height.Uint64()+1 ||
		!bytes.Equal(h.ParentHash, parent.Hash()) {
		return ErrNotContiguous
	}
	if h.Time <= parent.Time ||
		h.Difficulty.Cmp(CalcDifficulty(h.Time, parent)) != 0 {
		return ErrInvalidBlock
	}
	return nil
}

type Headers []*Header

func (rs Headers) Hash() []byte {
	hl := rs.hashList()
	return merkle.HashFromByteSlices(hl)
}

func (rs Headers) hashList() [][]byte {
	hl := make([][]byte, len(rs))
	for i := 0; i < len(rs); i++ {
		hl[i] = rs[i].Hash()
	}
	return hl
}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)

	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072) // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)

func CalcDifficulty(time uint64, parent *Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	// diff = (parent_diff +
	//         (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
	//        ) + 2^(periodCount - 2)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x := new(big.Int).SetUint64(time - parent.Time)
	x.Div(x, big10)

	if common.BytesToHash(parent.UncleHash.Bytes()) == types.EmptyUncleHash {
		x.Sub(big1, x)
	} else {
		x.Sub(big2, x)
	}

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	y := new(big.Int)
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(MinimumDifficulty) < 0 {
		x.Set(MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Height, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}
