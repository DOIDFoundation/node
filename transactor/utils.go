package transactor

import (
	"bytes"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func ValidateDoidNameSignatrue(doidName string, singer types.Address, signature []byte) bool {
	chainId := "1"
	message := append([]byte(chainId), []byte(doidName)...)
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
