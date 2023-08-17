package tx

import "github.com/DOIDFoundation/node/types"

// update doidname from exist owner to input owner
type Update struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	Signature types.Hash    `json:"signature"`
}

func (u *Update) Type() Type {
	return TypeUpdate
}
