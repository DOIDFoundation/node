package store

import (
	"math"

	"github.com/DOIDFoundation/node/types"
)

func (bs *BlockStore) ReadHashByHeight(height uint64) types.Hash {
	hash, err := bs.db.Get(headerHashKey(height))
	if err != nil {
		bs.Logger.Error("failed to read hash by height", "err", err, "height", height)
		return nil
	}

	if len(hash) == 0 {
		return nil
	}
	return hash
}

// GetHashByHeight stores hash by height for the head chain
func (bs *BlockStore) WriteHashByHeight(height uint64, hash types.Hash) {
	if err := bs.db.Set(headerHashKey(height), hash); err != nil {
		bs.Logger.Error("failed to store header hash by height", "err", err, "height", height, "hash", hash)
		panic(err)
	}
}

func (bs *BlockStore) DeleteHashByHeight(height uint64) {
	if err := bs.db.Delete(headerHashKey(height)); err != nil {
		bs.Logger.Error("failed to delete header hash by height", "err", err, "height", height)
		panic(err)
	}

	flag := bs.sqlitedb.RemoveMinerByHeight(height)
	if flag == false {
		bs.Logger.Error("Failed to remove miner")
		panic("Failed to remove miner")
	}
}

func (bs *BlockStore) DeleteHashByHeightFrom(height uint64) {
	iter, err := bs.db.Iterator(headerHashKey(height), headerHashKey(math.MaxUint64))
	if err != nil {
		bs.Logger.Error("failed to delete header hashes since height", "err", err, "height", height)
		panic(err)
	}
	defer iter.Close()

	b := bs.db.NewBatch()
	defer b.Close()

	for iter.Valid() {
		key := iter.Key()
		b.Delete(key)
		iter.Next()
	}

	if err = iter.Error(); err != nil {
		bs.Logger.Error("failed to delete header hashes since height", "err", err, "height", height)
		panic(err)
	}

	if err = b.WriteSync(); err != nil {
		bs.Logger.Error("failed to delete header hashes since height", "err", err, "height", height)
		panic(err)
	}
}

func (bs *BlockStore) ReadHeadBlockHash() types.Hash {
	hash, err := bs.db.Get(headBlockKey)
	if err != nil {
		bs.Logger.Error("failed to read head block hash", "err", err)
		return nil
	}
	if len(hash) == 0 {
		return nil
	}
	return hash
}

func (bs *BlockStore) WriteHeadBlockHash(hash types.Hash) {
	if err := bs.db.Set(headBlockKey, hash.Bytes()); err != nil {
		bs.Logger.Error("Failed to store last block's hash", "err", err, "hash", hash)
		panic(err)
	}
}

func (bs *BlockStore) ReadHeadBlock() *types.Block {
	headBlockHash := bs.ReadHeadBlockHash()
	if headBlockHash == nil {
		return nil
	}
	height := bs.ReadHeightByHash(headBlockHash)
	if height == nil {
		return nil
	}
	return bs.ReadBlock(*height, headBlockHash)
}
