package core

import (
	"math/big"

	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/types"
)

type API struct {
	chain *BlockChain
}

func RegisterAPI(chain *BlockChain) {
	rpc.RegisterName("doid", &API{chain: chain})
}

func (a *API) GetBlockByHeight(height uint64) *types.Block {
	return a.chain.BlockByHeight(height)
}

func (a *API) GetBlockByHash(hash types.Hash) *types.Block {
	return a.chain.blockStore.ReadBlock(*a.chain.blockStore.ReadHeightByHash(hash), hash)
}

func (a *API) GetBlockTD(hash types.Hash) *big.Int {
	return a.chain.blockStore.ReadTd(*a.chain.blockStore.ReadHeightByHash(hash), hash)
}

func (a *API) CurrentHeight() uint64 {
	return a.CurrentBlock().Header.Height.Uint64()
}

func (a *API) CurrentBlock() *types.Block {
	return a.chain.LatestBlock()
}

func (a *API) CurrentTD() *big.Int {
	return a.chain.GetTd()
}
