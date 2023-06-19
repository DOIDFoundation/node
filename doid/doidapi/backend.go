package doidapi

import (
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/types"
)

type Backend interface{
	SendTransaction(tx *types.Tx) error
	MemPool() *mempool.Mempool
	BlockChain() *core.BlockChain
}