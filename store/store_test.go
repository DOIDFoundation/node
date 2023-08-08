package store_test

import (
	"os"
	"testing"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/store"
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
