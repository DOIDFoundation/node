package core

import (
	"bytes"
	"container/list"
	"math/big"

	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	lru "github.com/hashicorp/golang-lru"
)

type lightHeader struct {
	Height uint64
	Hash   types.Hash
	Miner  types.Address
}

// used to store a sort of contiguous block headers for later applying
type HeaderChain struct {
	Logger log.Logger
	store  *store.BlockStore

	headers []lightHeader
	last    *types.Header
	td      *big.Int

	blockCache *lru.Cache // Cache for the most recent blockes
}

func newHeaderChain(store *store.BlockStore, logger log.Logger) *HeaderChain {
	blockCache, _ := lru.New(256)
	return &HeaderChain{
		store:      store,
		Logger:     logger.With("module", "blockchain"),
		blockCache: blockCache,
	}
}

func (hc *HeaderChain) CanStartFrom(height uint64, hash types.Hash) bool {
	headers := list.New()
	for {
		// append stored blocks when we found an ancestor in our chain
		if hc.store.ReadTd(height, hash) != nil || (hc.last != nil && hc.last.Height.Uint64() == height) {
			for e := headers.Front(); e != nil; e = e.Next() {
				header := e.Value.(*lightHeader)
				block := hc.store.ReadBlock(header.Height, header.Hash)
				if err := hc.AppendBlocks([]*types.Block{block}); err != nil {
					// can not append stored blocks
					return false
				}
			}
			return true
		}
		// check if we have block in store
		header := hc.store.ReadHeader(height, hash)
		if header == nil {
			return false
		}
		// check if we have parents of this block
		headers.PushFront(&lightHeader{header.Height.Uint64(), header.Hash(), header.Miner})
		height = header.Height.Uint64() - 1
		hash = header.ParentHash
	}
}

func (hc *HeaderChain) getBlock(height uint64, hash types.Hash) *types.Block {
	if cached, ok := hc.blockCache.Get(hash.String()); ok {
		block := cached.(*types.Block)
		return block
	}
	return hc.store.ReadBlock(height, hash)
}

func (hc *HeaderChain) saveAndCacheBlock(block *types.Block) {
	hc.store.WriteBlock(block)
	hc.blockCache.Add(block.Hash().String(), block)
}

// append blocks to calculate total difficulty, only contiguous and valid blocks are included
func (hc *HeaderChain) AppendBlocks(blocks []*types.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// save block and calculate total difficulty for contiguous blocks
	for _, block := range blocks {
		height, hash := block.Header.Height.Uint64(), block.Hash()
		if hc.last == nil {
			if bytes.Equal(hc.store.ReadHashByHeight(height), hash) {
				// skip blocks already in head chain
				continue
			}

			hc.last = hc.store.ReadHeader(height-1, block.Header.ParentHash)
			hc.td = hc.store.ReadTd(height-1, block.Header.ParentHash)
			if hc.td == nil {
				return ErrUnknownAncestor
			}
		}
		if err := block.Header.IsValid(hc.last); err != nil {
			hc.Logger.Error("invalid block", "header", block.Header, "prev", hc.last, "hash", hc.last.Hash(), "err", err)
			return err
		}
		hc.td.Add(hc.td, block.Header.Difficulty)
		hc.saveAndCacheBlock(block)
		hc.headers = append(hc.headers, lightHeader{Height: height, Hash: hash, Miner: block.Header.Miner})
		hc.last = block.Header
	}
	return nil
}

func (hc *HeaderChain) GetTd() *big.Int {
	return hc.td
}
