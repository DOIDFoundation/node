package network

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
)

type version struct {
	//Version  byte
	Height uint64
	Td     *big.Int
	ID     string
}

func (v version) serialize() []byte {
	bz, err := rlp.EncodeToBytes(v)
	if err != nil {
		panic(err)
	}
	return bz
}

func (v *version) deserialize(d []byte) {
	err := rlp.DecodeBytes(d, v)
	if err != nil {
		panic(err)
	}
}
