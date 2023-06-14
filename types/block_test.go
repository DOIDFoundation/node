package types_test

import (
	"testing"
	"time"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	h := &types.Header{
		ParentHash: []byte{0},
		Miner:      []byte{0},
		Root:       []byte{0},
		TxHash:     []byte{0},
		Difficulty: common.Big0,
		Height:     common.Big0,
		Time:       time.Now(),
		Extra:      []byte{0},
		Nonce:      types.EncodeNonce(0),
	}
	hash := h.Hash()
	hCopy := types.CopyHeader(h)
	hCopyHash := hCopy.Hash()
	assert.Equal(t, hash, hCopyHash)

	hCopy.ParentHash[0] = 1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Miner[0] = 1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Root[0] = 1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.TxHash[0] = 1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Difficulty = common.Big1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Height = common.Big1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Extra[0] = 1
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)

	hCopy = types.CopyHeader(h)
	hCopy.Nonce = types.EncodeNonce(1)
	hCopyHash = hCopy.Hash()
	assert.NotEqual(t, hash, hCopyHash)
}

func TestBlock(t *testing.T) {
	h := &types.Header{}
	b := types.NewBlockWithHeader(h)
	assert.Nil(t, b.Header.TxHash)
	b.Hash()
	assert.Equal(t, hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"), b.Header.TxHash.Bytes())
}
