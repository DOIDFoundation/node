package doid

import (
	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/mempool"
	"github.com/DOIDFoundation/node/node"
	"github.com/DOIDFoundation/node/rpc"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
)

type Doid struct {
	mp *mempool.Mempool
	blockChain *core.BlockChain

	APIBackend *DoidAPIBackend
}

func New(node *node.Node) (*Doid , error){
	d := &Doid{}
	d.mp = mempool.NewMempool(node.Chain(), node.Logger)
	d.blockChain = node.Chain()

	d.APIBackend = &DoidAPIBackend{Doid: d}

	// add Apis
	api := &ethrpc.API{Namespace: "doid", Version: "1.0", Service: NewPublicTransactionPoolAPI(d.APIBackend), Public: true}
	rpc.APIs = append(rpc.APIs, *api)

	return d, nil
}

func (d *Doid) MemPool() *mempool.Mempool {return d.mp}
func (d *Doid) BlockChain() *core.BlockChain {return d.blockChain}

func (d *Doid) StartMining() {

}