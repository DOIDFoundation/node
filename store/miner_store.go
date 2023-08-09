package store

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
)

type MinerStore interface {
	Close()
	AddMiner(height uint64, miner types.Address) bool
	RemoveMinerByHeight(height uint64) bool
	RemoveMinerFromHeight(height uint64) bool
	CountBlockByMiner(miner types.Address, limit int) int
	QueryBlockByMiner(miner types.Address, limit int, page int) []uint64
}

type MinerStoreNop struct {
}

var newMinerStore = func(log.Logger) MinerStore {
	return &MinerStoreNop{}
}

func (s *MinerStoreNop) Close() {

}

func (*MinerStoreNop) AddMiner(height uint64, miner types.Address) bool {
	return true
}

func (*MinerStoreNop) RemoveMinerByHeight(height uint64) bool {
	return true
}

func (*MinerStoreNop) RemoveMinerFromHeight(height uint64) bool {
	return true
}

func (*MinerStoreNop) CountBlockByMiner(miner types.Address, limit int) int {
	return 0
}

func (*MinerStoreNop) QueryBlockByMiner(miner types.Address, limit int, page int) []uint64 {
	return nil
}
