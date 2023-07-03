package store

import (
	"bytes"
	"encoding/binary"
	"math/big"
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
	log.Logger
	db cmtdb.DB
}

func NewBlockStore(logger log.Logger) (*BlockStore, error) {
	homeDir := viper.GetString(flags.Home)
	db, err := cmtdb.NewDB("chaindata", cmtdb.BackendType(viper.GetString(flags.DB_Engine)), filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	return &BlockStore{
		Logger: logger.With("module", "blockStore"),
		db:     db,
	}, nil
}

func (bs *BlockStore) ReadBlock(height uint64, hash types.Hash) *types.Block {
	header := bs.ReadHeader(height, hash)
	if header == nil {
		return nil
	}
	// data := bs.ReadData(hash)
	// if data == nil {
	// 	return nil
	// }
	return types.NewBlockWithHeader(header)
}

func (bs *BlockStore) WriteBlock(block *types.Block) {
	// bs.WriteData(block.Data)
	block.Hash() // call hash to fill header if needed
	bs.WriteHeader(block.Header)
}

// ReadHeader retrieves the block header corresponding to the hash.
func (bs *BlockStore) ReadHeader(height uint64, hash types.Hash) *types.Header {
	bz, err := bs.db.Get(headerKey(height, hash))
	if err != nil {
		bs.Logger.Error("failed to read block header", "err", err, "height", height, "hash", hash)
		return nil
	}

	if len(bz) == 0 {
		return nil
	}

	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(bz), header); err != nil {
		bs.Logger.Error("Invalid block header RLP", "err", err, "height", height, "hash", hash)
		return nil
	}
	return header
}

// WriteHeader writes the block header corresponding to the hash.
func (bs *BlockStore) WriteHeader(header *types.Header) {
	var (
		hash   = header.Hash()
		height = header.Height.Uint64()
	)

	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		bs.Logger.Error("failed to RLP encode header", "err", err)
		panic(err)
	}
	if err := bs.db.Set(headerKey(height, hash), data); err != nil {
		bs.Logger.Error("failed to store header by hash", "err", err)
		panic(err)
	}
	bs.WriteHeightByHash(hash, height)
}

// ReadDataRLP retrieves the block body (transactions and uncles) in RLP encoding.
func (bs *BlockStore) ReadDataRLP(hash types.Hash) rlp.RawValue {
	bz, err := bs.db.Get(hash)
	if err != nil {
		bs.Logger.Error("failed to read block data", "err", err)
		return nil
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
		bs.Logger.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

func (bs *BlockStore) WriteData(data *cmttypes.Data) {
	bs.Logger.Error("not implemented")
}

// ReadHeightByHash returns the header height assigned to a hash.
func (bs *BlockStore) ReadHeightByHash(hash types.Hash) *uint64 {
	data, err := bs.db.Get(headerHeightKey(hash))
	if err != nil {
		bs.Logger.Error("failed to read height by hash", "err", err, "hash", hash)
		panic(err)
	}
	if len(data) != 8 {
		return nil
	}
	height := binary.BigEndian.Uint64(data)
	return &height
}

// WriteHeightByHash stores the hash->height mapping.
func (bs *BlockStore) WriteHeightByHash(hash types.Hash, height uint64) {
	key := headerHeightKey(hash)
	enc := encodeBlockHeight(height)
	if err := bs.db.Set(key, enc); err != nil {
		bs.Logger.Error("Failed to store hash to height mapping", "err", err)
		panic(err)
	}
}

// DeleteHeaderHeight removes hash->height mapping.
func (bs *BlockStore) DeleteHeaderHeight(hash types.Hash) {
	if err := bs.db.Delete(headerHeightKey(hash)); err != nil {
		bs.Logger.Error("Failed to delete hash to height mapping", "err", err)
		panic(err)
	}
}

// ReadTd retrieves a block's total difficulty corresponding to the hash.
func (bs *BlockStore) ReadTd(height uint64, hash types.Hash) *big.Int {
	data, err := bs.db.Get(headerTDKey(height, hash))
	if err != nil {
		bs.Logger.Error("Failed to read block total difficulty", "err", err)
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		bs.Logger.Error("Invalid block total difficulty RLP", "hash", hash, "err", err)
		return nil
	}
	return td
}

// WriteTd stores the total difficulty of a block into the database.
func (bs *BlockStore) WriteTd(height uint64, hash types.Hash, td *big.Int) {
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		bs.Logger.Error("Failed to RLP encode block total difficulty", "err", err)
		panic(err)
	}
	if err := bs.db.Set(headerTDKey(height, hash), data); err != nil {
		bs.Logger.Error("Failed to store block total difficulty", "err", err)
		panic(err)
	}
}

// DeleteTd removes all block total difficulty data associated with a hash.
func (bs *BlockStore) DeleteTd(height uint64, hash types.Hash) {
	if err := bs.db.Delete(headerTDKey(height, hash)); err != nil {
		bs.Logger.Error("Failed to delete block total difficulty", "err", err)
		panic(err)
	}
}

func (bs *BlockStore) Close() error {
	bs.Logger.Debug("closing block store")
	return bs.db.Close()
}
