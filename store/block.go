package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"

	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/viper"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type BlockStore struct {
	log.Logger
	db       cmtdb.DB
	sqlitedb *SqliteStore
}

type SqliteStore struct {
	log.Logger
	Db     *sql.DB
	dbpath string
}

type DBStore interface {
	Init(path string) bool
	AddMiner(height uint64, miner types.Address) bool
	RemoveMinerByHeight(height uint64) bool
	QueryBlockByMiner(miner types.Address) []uint64
	CountBlockByMiner(miner types.Address) int
}

func (s *SqliteStore) Init(sqlite3DbPath string) bool {
	s.Logger.Info("SQLITE Init")
	// homeDir := viper.GetString(flags.Home)
	// sqlite3DbPath := filepath.Join(homeDir, "data", "sqlite3.db")
	sqlitedb, err := sql.Open("sqlite3", sqlite3DbPath)

	if err != nil {
		s.Logger.Error("SQL.OPEN", err.Error())
		return false
	}
	minerTableDDL := "CREATE TABLE IF NOT EXISTS miner_block (id INTEGER PRIMARY KEY, miner_address TEXT, block_height INTEGER)"
	_, err = sqlitedb.Exec(minerTableDDL)
	if err != nil {
		s.Logger.Error("EXEC", err.Error())
		return false
	}
	s.Db = sqlitedb
	s.dbpath = sqlite3DbPath
	return true
}

func (s *SqliteStore) AddMiner(height uint64, miner types.Address) bool {
	var exists bool
	err := s.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM miner_block WHERE miner_address=? and block_height=?)", hexutil.Encode(miner.Bytes()), height).Scan(&exists)
	if err != nil {
		s.Logger.Error("AddMiner Query error:", err.Error())
		return false
	}

	if exists {
		s.Logger.Info("AddMiner Record exists")
		return true
	} else {
		_, err := s.Db.Exec("INSERT INTO miner_block (miner_address,block_height) VALUES (?,?)", hexutil.Encode(miner.Bytes()), height)
		if err != nil {
			s.Logger.Error("AddMiner INSERT error:", err.Error())
			return false
		}
		s.Logger.Info("SQLITE AddMiner success")
		return true
	}

}

func (s *SqliteStore) RemoveMinerByHeight(height uint64) bool {
	_, err := s.Db.Exec("DELETE FROM miner_block where block_height=?", height)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		s.Logger.Error("RemoveMinerByHeight error:", err.Error())
		return false
	}

	return true
}

func (s *SqliteStore) CountBlockByMiner(miner types.Address, limit int) int {

	var totalRecords int
	err := s.Db.QueryRow("SELECT COUNT(*) FROM miner_block where miner_address=?", hexutil.Encode(miner.Bytes())).Scan(&totalRecords)
	if err != nil {
		s.Logger.Error("CountBlockByMiner query_count error:", err.Error())
		return 0
	}
	return totalRecords
}

func (s *SqliteStore) QueryBlockByMiner(miner types.Address, limit int, page int) []uint64 {

	offset := (page - 1) * limit
	rows, err := s.Db.Query("SELECT block_height FROM miner_block where miner_address=? order by block_height asc LIMIT ? OFFSET ?", hexutil.Encode(miner.Bytes()), limit, offset)
	if err != nil {
		s.Logger.Error("QueryBlockByMiner query_record error:", err.Error())
		return nil
	}

	var k []uint64
	for rows.Next() {
		var height int
		err = rows.Scan(&height)
		if err != nil {
			s.Logger.Error("Scan", err)
			return nil
		}
		k = append(k, uint64(height))
	}
	rows.Close()
	return k
}

func NewBlockStore(logger log.Logger) (*BlockStore, error) {
	homeDir := viper.GetString(flags.Home)
	db, err := cmtdb.NewDB("chaindata", cmtdb.BackendType(viper.GetString(flags.DB_Engine)), filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}

	_sqliteDb := &SqliteStore{Logger: logger.With("module", "sqliteStore")}
	initRet := _sqliteDb.Init(filepath.Join(homeDir, "sqlite3.db"))
	logger.Info("sqlite_store init:", initRet)

	return &BlockStore{
		Logger:   logger.With("module", "blockStore"),
		db:       db,
		sqlitedb: _sqliteDb,
	}, nil
}

func (bs *BlockStore) ReadBlock(height uint64, hash types.Hash) *types.Block {
	header := bs.ReadHeader(height, hash)
	if header == nil {
		return nil
	}
	uncles := new(types.Headers)
	if err := bs.ReadData(unclesKey(header.UncleHash), uncles); err != nil {
		bs.Logger.Error("invalid block uncles", "err", err, "hash", header.UncleHash)
		return nil
	}
	txs := new(types.Txs)
	if err := bs.ReadData(txsKey(header.TxHash), txs); err != nil {
		bs.Logger.Error("invalid block txs", "err", err, "hash", header.TxHash)
		return nil
	}
	receipts := new(types.Receipts)
	if err := bs.ReadData(receiptsKey(header.ReceiptHash), receipts); err != nil {
		bs.Logger.Error("invalid block receipts", "err", err, "hash", header.ReceiptHash)
		return nil
	}
	block := types.NewBlockWithHeader(header)
	block.Txs = *txs
	block.Receipts = *receipts
	block.Uncles = *uncles
	return block
}

func (bs *BlockStore) WriteBlock(block *types.Block) {
	bs.WriteData(txsKey(block.Txs.Hash()), block.Txs)
	for _, t := range block.Txs {
		bs.WriteData(transactionKey(t.Hash()), t)
	}
	bs.WriteData(unclesKey(block.Uncles.Hash()), block.Uncles)
	bs.WriteData(receiptsKey(block.Receipts.Hash()), block.Receipts)
	for i, r := range block.Receipts {
		bs.WriteData(
			receiptKey(r.TxHash),
			types.StoredReceipt{
				Receipt:          *r,
				BlockHash:        block.Hash(),
				BlockNumber:      block.Header.Height,
				TransactionIndex: uint(i),
			},
		)
	}
	block.Hash() // call hash to fill header if needed
	bs.WriteHeader(block.Header)
}

// ReadHeader retrieves the block header corresponding to the hash.
func (bs *BlockStore) ReadHeader(height uint64, hash types.Hash) *types.Header {
	header := new(types.Header)
	if err := bs.ReadData(headerKey(height, hash), header); err != nil {
		bs.Logger.Error("invalid block header", "err", err, "height", height, "hash", hash)
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
	bs.WriteData(headerKey(height, hash), header)
	bs.WriteHeightByHash(hash, height, header.Miner)
}

// ReadDataRLP retrieves the data in RLP encoding.
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

// ReadData retrieves the data corresponding to the hash and decode into result.
func (bs *BlockStore) ReadData(hash types.Hash, result interface{}) error {
	data := bs.ReadDataRLP(hash)
	if len(data) == 0 {
		return errors.New("empty data read")
	}
	return rlp.DecodeBytes(data, result)
}

func (bs *BlockStore) WriteData(hash types.Hash, data interface{}) {
	// Write the encoded data
	bz, err := rlp.EncodeToBytes(data)
	if err != nil {
		bs.Logger.Error("failed to RLP encode header", "err", err)
		panic(err)
	}
	if err := bs.db.Set(hash, bz); err != nil {
		bs.Logger.Error("failed to store header by hash", "err", err)
		panic(err)
	}
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
func (bs *BlockStore) WriteHeightByHash(hash types.Hash, height uint64, miner types.Address) {
	key := headerHeightKey(hash)
	enc := encodeBlockHeight(height)
	if err := bs.db.Set(key, enc); err != nil {
		bs.Logger.Error("Failed to store hash to height mapping", "err", err)
		panic(err)
	}

	flag := bs.sqlitedb.AddMiner(height, miner)
	if flag == false {
		bs.Logger.Error("Failed to store miner to sqlite3")
		panic("Failed to store miner to sqlite3")
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

func (bs *BlockStore) ReadTx(hash types.Hash) (result types.Tx) {
	if err := bs.ReadData(transactionKey(hash), &result); err != nil {
		bs.Logger.Error("Failed to read tx", "err", err, "hash", hash)
		return nil
	}
	return
}

func (bs *BlockStore) ReadReceipt(hash types.Hash) (result *types.StoredReceipt) {
	result = new(types.StoredReceipt)
	if err := bs.ReadData(receiptKey(hash), result); err != nil {
		bs.Logger.Error("Failed to read receipt", "err", err, "hash", hash)
		return nil
	}
	return
}

func (bs *BlockStore) Close() error {
	bs.Logger.Debug("closing block store")
	return bs.db.Close()
}

func (bs *BlockStore) GetSqliteDB() *SqliteStore {
	return bs.sqlitedb
}
