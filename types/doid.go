package types

import (
	"github.com/ethereum/go-ethereum/crypto"
)

func DOIDHash(doid string) Hash {
	return crypto.Keccak256([]byte(doid))
}
