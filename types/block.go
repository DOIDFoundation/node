package types

import (
	"math/big"
	"time"
)

type Block struct {
	Header *Header `json:"header"`
	Data   `json:"data"`

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
	if b.Header.TxHash == nil {
		b.Header.TxHash = b.Data.Hash()
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
	ParentHash Hash       `json:"parentHash"       gencodec:"required"`
	Miner      Address    `json:"miner"            gencodec:"required"`
	Root       Hash       `json:"stateRoot"        gencodec:"required"`
	TxHash     Hash       `json:"transactionsRoot" gencodec:"required"`
	Difficulty *big.Int   `json:"difficulty"       gencodec:"required"`
	Height     *big.Int   `json:"height"           gencodec:"required"`
	Time       time.Time  `json:"timestamp"        gencodec:"required"`
	Extra      []byte     `json:"extraData"        gencodec:"required"`
	Nonce      BlockNonce `json:"nonce"`
}

func (h *Header) Hash() Hash {
	return rlpHash(h)
}
