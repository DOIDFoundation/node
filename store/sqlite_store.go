package store

import (
	"database/sql"

	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type SqliteStore struct {
	log.Logger
	Db     *sql.DB
	dbpath string
}

func (s *SqliteStore) Init(sqlite3DbPath string) bool {
	s.Logger.Info("SQLITE Init")
	// homeDir := viper.GetString(flags.Home)
	// sqlite3DbPath := filepath.Join(homeDir, "data", "sqlite3.db")
	sqlitedb, err := sql.Open("sqlite3", sqlite3DbPath)

	if err != nil {
		s.Logger.Error("SQL.OPEN", "err", err)
		return false
	}
	minerTableDDL := "CREATE TABLE IF NOT EXISTS miner_block (id INTEGER PRIMARY KEY, miner_address TEXT, block_height INTEGER)"
	_, err = sqlitedb.Exec(minerTableDDL)
	if err != nil {
		s.Logger.Error("EXEC", "err", err)
		return false
	}
	s.Db = sqlitedb
	s.dbpath = sqlite3DbPath
	return true
}

func (s *SqliteStore) AddMiner(height uint64, miner types.Address) bool {
	result, err := s.Db.Exec("INSERT INTO miner_block (miner_address,block_height) VALUES (?,?)", hexutil.Encode(miner.Bytes()), height)
	if err != nil {
		s.Logger.Error("AddMiner INSERT error:", "err", err)
		return false
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		s.Logger.Debug("AddMiner Record exists")
		return true
	} else if err != nil {
		s.Logger.Error("AddMiner INSERT error:", "err", err)
		return false
	}
	// s.Logger.Info("SQLITE AddMiner success")
	return true
}

func (s *SqliteStore) RemoveMinerByHeight(height uint64) bool {
	_, err := s.Db.Exec("DELETE FROM miner_block where block_height=?", height)
	if err != nil {
		s.Logger.Error("RemoveMinerByHeight error:", "err", err)
		return false
	}

	return true
}

func (s *SqliteStore) RemoveMinerFromHeight(height uint64) bool {
	_, err := s.Db.Exec("DELETE FROM miner_block where block_height>=?", height)
	if err != nil {
		s.Logger.Error("RemoveMinerFromHeight error:", "err", err)
		return false
	}

	return true
}

func (s *SqliteStore) CountBlockByMiner(miner types.Address, limit int) int {

	var totalRecords int
	err := s.Db.QueryRow("SELECT COUNT(*) FROM miner_block where miner_address=?", hexutil.Encode(miner.Bytes())).Scan(&totalRecords)
	if err != nil {
		s.Logger.Error("CountBlockByMiner query_count error:", "err", err)
		return 0
	}
	return totalRecords
}

func (s *SqliteStore) QueryBlockByMiner(miner types.Address, limit int, page int) []uint64 {

	offset := (page - 1) * limit
	rows, err := s.Db.Query("SELECT block_height FROM miner_block where miner_address=? order by block_height asc LIMIT ? OFFSET ?", hexutil.Encode(miner.Bytes()), limit, offset)
	if err != nil {
		s.Logger.Error("QueryBlockByMiner query_record error:", "err", err)
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
