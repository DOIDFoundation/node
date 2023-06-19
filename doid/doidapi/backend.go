package doidapi

import (
	"github.com/DOIDFoundation/node/types"
)

type Backend interface{
	SendTransaction(tx *types.Tx) error
}