package tx

import "github.com/DOIDFoundation/node/types"

type Update struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	Signature types.Hash    `json:"signature"`
}

func (r *Update) Type() Type {
	return TypeUpdate
}
