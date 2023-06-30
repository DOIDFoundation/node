package core

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	lru "github.com/hashicorp/golang-lru"
)

type lightHeader struct {
	Height uint64
	Hash   types.Hash
}

// used to store a sort of contiguous block headers for later applying
type HeaderChain struct {
	Logger log.Logger
	store  *store.BlockStore

	headers []lightHeader
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

	first := blocks[0]

	if len(hc.headers) > 0 {
		last := hc.headers[len(hc.headers)-1]
		if first.Header.Height.Uint64() != last.Height+1 && !bytes.Equal(first.Header.ParentHash, last.Hash) {
			hc.Logger.Error("Non contiguous block insert", "height", first.Header.Height, "hash", first.Hash(), "parent", first.Header.ParentHash, "last", last)
			return errors.New("blocks not contiguous")
		}
	} else {
		hc.td = hc.store.ReadTd(first.Header.Height.Uint64()-1, first.Header.ParentHash)
		if hc.td == nil {
			return ErrUnknownAncestor
		}
	}

	parent := hc.store.ReadHeader(first.Header.Height.Uint64()-1, first.Header.ParentHash)
	if !first.Header.IsValid(parent) {
		hc.Logger.Error("invalid block", "header", first.Header, "last", parent, "hash", parent.Hash())
		return errors.New("invalid block")
	}

	// save total difficulty for contiguous blocks
	var prev *types.Block
	for _, block := range blocks {
		height := block.Header.Height.Uint64()
		if prev != nil && !block.Header.IsValid(prev.Header) {
			hc.Logger.Error("invalid block", "header", block.Header, "prev", prev.Header, "hash", prev.Hash())
			return errors.New("invalid block")
		}
		hash := block.Hash()
		hc.td.Add(hc.td, block.Header.Difficulty)
		hc.saveAndCacheBlock(block)
		hc.headers = append(hc.headers, lightHeader{Height: height, Hash: hash})
		prev = block
	}
	return nil
}

func (hc *HeaderChain) GetTd() *big.Int {
	return hc.td
}
