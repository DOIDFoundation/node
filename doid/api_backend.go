package doid

import (
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/types"
)


type DoidAPIBackend struct {
	Doid *Doid
}




func (d *DoidAPIBackend) SendTransaction(tx *types.Tx) error{
	return d.MemPool().AddLocal(tx)
}

func (d *DoidAPIBackend) MemPool() *mempool.Mempool{return d.Doid.MemPool()}
func (d *DoidAPIBackend) Chain() *core.BlockChain{return d.Doid.BlockChain()}
// func (d *DoidAPIBackend) StateStore() *cosmosdb.cosmosdb{return d.StateStore()}