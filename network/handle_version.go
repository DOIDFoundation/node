package network

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
)

type peerState struct {
	//Version  byte
	Height uint64
	Td     *big.Int
	ID     string
}

func (v peerState) serialize() []byte {
	bz, err := rlp.EncodeToBytes(v)
	if err != nil {
		panic(err)
	}
	return bz
}

func (v *peerState) deserialize(d []byte) {
	err := rlp.DecodeBytes(d, v)
	if err != nil {
		panic(err)
	}
}
