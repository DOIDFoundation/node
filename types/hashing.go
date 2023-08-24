package types

import (
	"sync"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// HashSize is the size of hash.
const HashSize = 32

type (
	Hash = cmtbytes.HexBytes

	// HashKey is the fixed length array key used as an index.
	HashKey = [HashSize]byte
)

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// rlpHash encodes x and hashes the encoded bytes.
func rlpHash(x interface{}) Hash {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	var h [HashSize]byte
	sha.Read(h[:])
	return h[:]
}

func HashToKey(hash Hash) HashKey {
	var k HashKey
	copy(k[:], hash)
	return k
}
