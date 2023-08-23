package doid

import (
	"encoding/hex"
	"errors"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/types"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
)

var hashMaxResult, _ = hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

func DOIDHash(doid string) types.Hash {
	return crypto.Keccak256([]byte(doid))
}

func OwnerHash(owner types.Address) types.Hash {
	return crypto.Keccak256([]byte(owner))
}

func OwnerStatePrefix() []byte {
	return []byte(":owner:")
}

func UpdateOwnerDOIDNames(tree *iavl.MutableTree, owner types.Address, name string, add bool) error {
	if !config.IsOwnerFork(tree.Version()) {
		return nil
	}
	nameHash := DOIDHash(name)
	nameBytes := []byte(name)
	if len(nameBytes) == 0 {
		return errors.New("invalid name to update")
	}
	ownerHash := OwnerHash(owner)
	ownerPrefix := OwnerStatePrefix()
	start := append(ownerHash, ownerPrefix...)

	key := append(start, nameHash...)
	if add {
		_, err := tree.Set(key, nameBytes)
		return err
	} else {
		tree.Remove(key)
	}
	return nil
}

func GetOwnerDOIDNames(tree *iavl.ImmutableTree, owner types.Address) ([][]byte, error) {
	names := [][]byte{}
	ownerHash := OwnerHash(owner)
	ownerPrefix := OwnerStatePrefix()
	start := append(ownerHash, ownerPrefix...)
	end := append(start, hashMaxResult...)
	iterator, err := tree.Iterator(start, end, true)
	if err != nil {
		return names, err
	}
	for {
		if !iterator.Valid() {
			break
		}
		names = append(names, iterator.Value())
		iterator.Next()
	}

	return names, nil
}
