package doid

import (
	"encoding/hex"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/types"
)

type DOIDApi struct {
	chain *core.BlockChain
}

type DOIDName struct {
	DOID  string     `json:"DOID"`
	Owner types.Hash `json:"owner"`
}

func (api *DOIDApi) GetOwner(params DOIDName) (string, error) {
	state, err := api.chain.LatestState()
	if err != nil {
		return "", err
	}
	owner, err := state.Get(types.DOIDHash(params.DOID))
	if err != nil {
		return "", err
	}
	ownerAddress := hex.EncodeToString(owner)
	return ownerAddress, nil
}

func RegisterAPI(chain *core.BlockChain) {
	rpc.RegisterName("doid", &DOIDApi{chain: chain})
	rpc.RegisterName("doid", &PublicTransactionPoolAPI{})
}
