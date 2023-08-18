package core

import "github.com/DOIDFoundation/node/store"

func (c *BlockChain) BlockStore() *store.BlockStore {
	return c.blockStore
}
