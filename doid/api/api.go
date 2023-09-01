package doidapi

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/doid"
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

// Status checks if a name is registered or available or reserved for registration
func (api *DOIDApi) Status(DOID string) (string, error) {
	state, err := api.chain.LatestState()
	if err != nil {
		return "", err
	}
	if owner, err := state.Get(doid.DOIDHash(DOID)); err != nil {
		return "", err
	} else if owner != nil {
		return "registered", nil
	}
	err = doid.ValidateDoidName(DOID, doid.ClassCNameLength)
	switch {
	case err == nil:
		return "available", nil
	case errors.Is(err, doid.ErrReserved):
		return "reserved", nil
	default:
		return "invalid", err
	}
}

func (api *DOIDApi) GetOwner(params DOIDName) (string, error) {
	state, err := api.chain.LatestState()
	if err != nil {
		return "", err
	}
	owner, err := state.Get(doid.DOIDHash(params.DOID))
	if err != nil {
		return "", err
	}
	ownerAddress := hex.EncodeToString(owner)
	return ownerAddress, nil
}

func (api *DOIDApi) GetOwnerDOIDNames(owner types.Hash) ([]string, error) {
	ret := []string{}
	state, err := api.chain.LatestState()
	if err != nil {
		return ret, err
	}
	names, _ := doid.GetOwnerDOIDNames(state, owner)
	for _, v := range names {
		ret = append(ret, fmt.Sprint(v))
	}
	return ret, nil
}

func RegisterAPI(chain *core.BlockChain) {
	rpc.RegisterName("doid", &DOIDApi{chain: chain})
	rpc.RegisterName("doid", &PublicTransactionPoolAPI{chain: chain})
}
