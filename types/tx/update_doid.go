package tx

import "github.com/DOIDFoundation/node/types"

type UpdateDOID struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	Signature types.Hash    `json:"signature"`
}

func (r *UpdateDOID) Type() Type {
	return TypeUpdateDOID
}
