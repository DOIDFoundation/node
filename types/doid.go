package types

import (
	"github.com/ethereum/go-ethereum/crypto"
)

func DOIDHash(doid string) Hash {
	return crypto.Keccak256([]byte(doid))
}

func OwnerHash(owner Address) Hash {
	return crypto.Keccak256([]byte(owner))
}

type OwnerState struct {
	OwnerHash Hash
	Names     [][]byte
}
