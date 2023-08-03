package store_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func newBlockStore(t *testing.T) *store.BlockStore {
	viper.SetDefault(flags.DB_Engine, "memdb")
	store, err := store.NewBlockStore(log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)
	return store
}

func TestSqliteStore(t *testing.T) {

	s := &store.SqliteStore{Logger: log.NewTMLogger(log.NewSyncWriter(os.Stdout))}
	require.NotNil(t, s)
	sqlite3DbPath := "./data.db"
	flag := s.Init(sqlite3DbPath)

	require.True(t, flag)

	// flag = s.AddMiner(1, types.HexToAddress("0x9a5de5673bb089924da48ca6fb3778766667dfe1"))
	// flag = s.RemoveMinerByHeight(1)
	// require.True(t, flag)

	ret := s.QueryBlockByMiner(types.HexToAddress("0x9a5de5673bb089924da48ca6fb3778766667dfe1"), 1, 1)
	for i, v := range ret {
		fmt.Printf("arr[%d] = %d\n", i, v)
	}
	require.NotNil(t, ret)

}
