package tx

import "github.com/DOIDFoundation/node/types"

type Reserve struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	Admin     types.Address `json:"admin"`
	Signature types.Hash    `json:"signature"`
}

func (r *Reserve) Type() Type {
	return TypeReserve
}
