package store

import (
	"encoding/binary"
	"fmt"

	"github.com/DOIDFoundation/node/types"
)

var (
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerTDSuffix     = []byte("t") // headerPrefix + num (uint64 big endian) + hash + headerTDSuffix -> td
	headerHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	headerHeightPrefix = []byte("H") // headerHeightPrefix + hash -> num (uint64 big endian)

	// headBlockKey tracks the latest known full block's hash.
	headBlockKey = []byte("LastBlock")
)

// encodeBlockHeight encodes a block height as big endian uint64
func encodeBlockHeight(height uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, height)
	return enc
}

// headerKey = headerPrefix + num (uint64 big endian) + hash
func headerKey(height uint64, hash types.Hash) []byte {
	return append(append(headerPrefix, encodeBlockHeight(height)...), hash.Bytes()...)
}

// headerTDKey = headerPrefix + num (uint64 big endian) + hash + headerTDSuffix
func headerTDKey(height uint64, hash types.Hash) []byte {
	return append(headerKey(height, hash), headerTDSuffix...)
}

// headerHashKey = headerPrefix + num (uint64 big endian) + headerHashSuffix
func headerHashKey(height uint64) []byte {
	return append(append(headerPrefix, encodeBlockHeight(height)...), headerHashSuffix...)
}

// headerHeightKey = headerHeightPrefix + hash
func headerHeightKey(hash types.Hash) []byte {
	return append(headerHeightPrefix, hash.Bytes()...)
}

func dataKey(height uint64) []byte {
	return []byte(fmt.Sprintf("D:%v", height))
}
