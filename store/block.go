package store

import (
	"bytes"
	"fmt"

	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"
)

type BlockStore struct {
	logger log.Logger
	db     cmtdb.DB
}

func NewBlockStore(db cmtdb.DB, logger log.Logger) *BlockStore {
	return &BlockStore{
		logger: logger.With("module", "blockStore"),
		db:     db,
	}
}

func (bs *BlockStore) ReadBlock(hash types.Hash, number int64) *types.Block {
	header := bs.ReadHeader(hash, number)
	if header == nil {
		return nil
	}
	data := bs.ReadData(hash, number)
	if data == nil {
		return nil
	}
	return types.NewBlockWithHeader(header)
}

// ReadHeaderRLP retrieves a block header in its raw RLP database encoding.
func (bs *BlockStore) ReadHeaderRLP(hash types.Hash, number int64) rlp.RawValue {
	bz, err := bs.db.Get(headerKey(number))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return nil
	}
	return bz
}

// ReadHeader retrieves the block header corresponding to the hash.
func (bs *BlockStore) ReadHeader(hash types.Hash, number int64) *types.Header {
	data := bs.ReadHeaderRLP(hash, number)
	if len(data) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		bs.logger.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// ReadDataRLP retrieves the block body (transactions and uncles) in RLP encoding.
func (bs *BlockStore) ReadDataRLP(hash types.Hash, number int64) rlp.RawValue {
	bz, err := bs.db.Get(dataKey(number))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return nil
	}
	return bz
}

// ReadData retrieves the block body corresponding to the hash.
func (bs *BlockStore) ReadData(hash types.Hash, number int64) *cmttypes.Data {
	data := bs.ReadDataRLP(hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(cmttypes.Data)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		bs.logger.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

func (bs *BlockStore) ReadHeadBlockHash() types.Hash {
	data, _ := bs.db.Get(headBlockKey)
	if len(data) == 0 {
		return types.Hash{}
	}
	return data
}

func (bs *BlockStore) WriteHeadBlockHash(hash types.Hash) {
	if err := bs.db.Set(headBlockKey, hash.Bytes()); err != nil {
		bs.logger.Error("Failed to store last block's hash", "err", err)
	}
}

func (bs *BlockStore) Close() error {
	bs.logger.Debug("closing block store")
	return bs.db.Close()
}

func headerKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func dataKey(height int64) []byte {
	return []byte(fmt.Sprintf("D:%v", height))
}

// headBlockKey tracks the latest known full block's hash.
var headBlockKey = []byte("HeadBlock")
