package store

import (
	"encoding/binary"

	"github.com/DOIDFoundation/node/types"
)

var (
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerTDSuffix     = []byte("t") // headerPrefix + num (uint64 big endian) + hash + headerTDSuffix -> td
	headerHashPrefix   = []byte("n") // headerHashPrefix + num (uint64 big endian) -> hash
	headerHeightPrefix = []byte("H") // headerHeightPrefix + hash -> num (uint64 big endian)
	txsPrefix          = []byte("T") // txsPrefix + hash -> txs
	unclesPrefix       = []byte("u") // unclesPrefix + hash -> uncles
	receiptsPrefix     = []byte("r") // receiptsPrefix + hash -> receipts

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
	return append(headerHashPrefix, encodeBlockHeight(height)...)
}

// headerHeightKey = headerHeightPrefix + hash
func headerHeightKey(hash types.Hash) []byte {
	return append(headerHeightPrefix, hash.Bytes()...)
}

// unclesKey = unclesPrefix + hash
func unclesKey(hash types.Hash) []byte {
	return append(unclesPrefix, hash.Bytes()...)
}

// receiptsKey = receiptsPrefix + hash
func receiptsKey(hash types.Hash) []byte {
	return append(receiptsPrefix, hash.Bytes()...)
}

// txsKey = txsPrefix + hash
func txsKey(hash types.Hash) []byte {
	return append(txsPrefix, hash.Bytes()...)
}
