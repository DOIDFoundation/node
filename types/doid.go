package types

import (
	"encoding/hex"
	"errors"

	"github.com/DOIDFoundation/node/config"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
)

var hashMaxResult, _ = hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

func DOIDHash(doid string) Hash {
	return crypto.Keccak256([]byte(doid))
}

func OwnerHash(owner Address) Hash {
	return crypto.Keccak256([]byte(owner))
}

func OwnerStatePrefix() []byte {
	return []byte(":owner:")
}

func UpdateOwnerDOIDNames(tree *iavl.MutableTree, owner Address, name string, add bool) error {
	if tree.Version() <= 104870 && config.NetworkID == 2 {
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

func GetOwnerDOIDNames(tree *iavl.ImmutableTree, owner Address) ([][]byte, error) {
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
