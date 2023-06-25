package store

import (
	"bytes"
	"fmt"
	"path/filepath"

	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/viper"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"
)

type BlockStore struct {
	logger log.Logger
	db     cmtdb.DB
}

func NewBlockStore(logger log.Logger) (*BlockStore, error) {
	homeDir := viper.GetString(flags.Home)
	db, err := cmtdb.NewDB("chaindata", cmtdb.BackendType(viper.GetString(flags.DB_Engine)), filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	return &BlockStore{
		logger: logger.With("module", "blockStore"),
		db:     db,
	}, nil
}

func (bs *BlockStore) ReadBlockByHeight(height uint64) *types.Block {
	hash, err := bs.db.Get(headerKey(height))
	if err != nil {
		panic(err)
	}

	if len(hash) == 0 {
		return nil
	}
	return bs.ReadBlock(hash)
}

func (bs *BlockStore) ReadBlock(hash types.Hash) *types.Block {
	header := bs.ReadHeader(hash)
	if header == nil {
		return nil
	}
	// data := bs.ReadData(hash)
	// if data == nil {
	// 	return nil
	// }
	return types.NewBlockWithHeader(header)
}

// ReadHeaderRLP retrieves a block header in its raw RLP database encoding.
func (bs *BlockStore) ReadHeaderRLP(hash types.Hash) rlp.RawValue {
	bz, err := bs.db.Get(hash)
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return nil
	}
	return bz
}

// ReadHeader retrieves the block header corresponding to the hash.
func (bs *BlockStore) ReadHeader(hash types.Hash) *types.Header {
	data := bs.ReadHeaderRLP(hash)
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
func (bs *BlockStore) ReadDataRLP(hash types.Hash) rlp.RawValue {
	bz, err := bs.db.Get(hash)
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return nil
	}
	return bz
}

// ReadData retrieves the block body corresponding to the hash.
func (bs *BlockStore) ReadData(hash types.Hash) *cmttypes.Data {
	data := bs.ReadDataRLP(hash)
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

func (bs *BlockStore) WriteBlock(block *types.Block) {
	// bs.WriteData(block.Data)
	bs.WriteHeader(block.Header)
}

// ReadHeader retrieves the block header corresponding to the hash.
func (bs *BlockStore) WriteHeader(header *types.Header) {
	var (
		hash   = header.Hash()
		height = header.Height.Uint64()
	)

	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		bs.logger.Error("failed to RLP encode header", "err", err, "height", height, "hash", hash)
		return
	}
	if err := bs.db.Set(hash, data); err != nil {
		bs.logger.Error("failed to store header by hash", "err", err, "height", height, "hash", hash)
	}
	if err := bs.db.Set(headerKey(height), hash); err != nil {
		bs.logger.Error("failed to store header hash by height", "err", err, "height", height, "hash", hash)
	}
}

func (bs *BlockStore) WriteData(data *cmttypes.Data) {
	bs.logger.Error("not implemented")
}

func (bs *BlockStore) ReadHeadBlockHash() types.Hash {
	data, _ := bs.db.Get(headBlockKey)
	if len(data) == 0 {
		return nil
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

func headerKey(height uint64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func dataKey(height uint64) []byte {
	return []byte(fmt.Sprintf("D:%v", height))
}

// headBlockKey tracks the latest known full block's hash.
var headBlockKey = []byte("HeadBlock")
