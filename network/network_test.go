package network

import (
	"os"
	"testing"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNetwork(t *testing.T) *Network {
	viper.SetDefault(flags.DB_Engine, "memdb")
	home, err := os.MkdirTemp("", "doid*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(home) })
	viper.Set(flags.Home, home)
	viper.SetDefault(flags.P2P_KeyFile, "p2p.key")
	viper.Set(flags.P2P_Addr, "/ip4/127.0.0.1/tcp/26667")
	chain, err := core.NewBlockChain(log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)
	n := NewNetwork(chain, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NotNil(t, n)
	return n
}

func TestLoadPrivateKey(t *testing.T) {
	n := newNetwork(t)
	viper.Set(flags.P2P_KeyFile, "doid.key")
	assert.Panics(t, func() { n.loadPrivateKey() })

	viper.Set(flags.P2P_KeyFile, "p2p.key")
	priv := n.loadPrivateKey()
	assert.NotNil(t, priv)

	viper.Set(flags.P2P_KeyFile, "doid.key")
	viper.Set(flags.P2P_Key, "8AjjYYqepD+OiodmZJynfJsIYpcV/Wi4vrZnzdhAGW37pgiWg0HSQ6/S3D8bGrbRzR5bkDjX6IpQymgr4bAV9A==")
	priv2 := n.loadPrivateKey()
	assert.NotNil(t, priv2)
	assert.NotEqual(t, priv, priv2)

	bz, err := priv.Raw()
	require.NoError(t, err)
	viper.Set(flags.P2P_Key, crypto.ConfigEncodeKey(bz))
	priv3 := n.loadPrivateKey()
	assert.NotNil(t, priv2)
	assert.Equal(t, priv, priv3)
	assert.NotEqual(t, priv2, priv3)
}
