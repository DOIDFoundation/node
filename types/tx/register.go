package tx

import "github.com/DOIDFoundation/node/types"

type Register struct {
	DOID      string        `json:"DOID"`
	Owner     types.Address `json:"owner"`
	NameHash  types.Hash 	`json:"nameHash"`
	Signature types.Hash    `json:"signature"`
}

func (r *Register) Type() types.TxType {
	return types.TxTypeRegister
}
