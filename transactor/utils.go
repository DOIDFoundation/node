package transactor

import (
	"bytes"
	"errors"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/types"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func updateOwnerState(tree *iavl.MutableTree, owner types.Address, name string, add bool) error {
	nameBytes := []byte(name)
	if len(nameBytes) == 0 {
		return errors.New("invalid name to update")
	}
	ownerHash := types.OwnerHash(owner)
	b, err := tree.Get(ownerHash)
	if err != nil {
		return err
	}
	ownerState := &types.OwnerState{}
	if b == nil {
		if add {
			ownerState.Names = [][]byte{nameBytes}
			ownerState.OwnerHash = ownerHash
		} else {
			return errors.New("owner doesn't have any name")
		}
	} else {
		err = rlp.DecodeBytes(b, ownerState)
		if err != nil {
			return err
		}
		if add {
			ownerState.Names = append(ownerState.Names, nameBytes)
		} else {
			names := [][]byte{}
			for i := 0; i < len(ownerState.Names); i++ {
				if bytes.Equal(ownerState.Names[i], nameBytes) {
					// ownerState.Names[i] = []byte(name)
					continue
				}
				names = append(names, ownerState.Names[i])
			}
			ownerState.Names = names
		}
	}
	encodedState, err := rlp.EncodeToBytes(ownerState)
	if err != nil {
		return err
	}
	tree.Set(ownerHash, encodedState)
	return nil
}

func ValidateDoidNameSignatrue(doidName string, singer types.Address, signature []byte) bool {
	message := append([]byte{config.NetworkID}, []byte(doidName)...)
	message = crypto.Keccak256(message, singer)
	recovered, err := crypto.SigToPub(message, signature)
	if err != nil {
		return false
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	return bytes.Equal(recoveredAddr.Bytes(), singer.Bytes())
}

func ValidateDoidName(s string, valid int) bool {
	length := getStringLength(s)
	if length <= valid {
		return false
	} else {
		return true
	}
}

func getStringLength(s string) int {
	var strLength int
	i := 0
	b := []byte(s)
	for strLength = 0; i < len(b); strLength++ {
		if b[i] < 0x80 {
			i++ // ascii code : 1byte
		} else {
			// utf-8 dynamic length
			strLength++
			if b[i] < 0xE0 {
				i += 2
			} else if b[i] < 0xF0 {
				i += 3
			} else if b[i] < 0xF8 {
				i += 4
			} else if b[i] < 0xFC {
				i += 5
			} else {
				i += 6
			}
		}
	}
	return strLength
}
