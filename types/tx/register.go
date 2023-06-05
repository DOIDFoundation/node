package tx

import "github.com/DOIDFoundation/node/types"

type Register struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	Signature types.Hash    `json:"signature"`
}

func (r *Register) Type() byte {
	return TypeRegister
}
