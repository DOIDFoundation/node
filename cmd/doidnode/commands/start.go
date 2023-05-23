package commands

import (
	"fmt"

	"github.com/DOIDFoundation/node/cmd/doidnode/app"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	cosmosdb "github.com/cosmos/cosmos-db"
)

func NewNode(config *cfg.Config, logger log.Logger) (*node.Node, error) {
	db, err := cosmosdb.NewDB("leveldb", cosmosdb.GoLevelDBBackend, config.DBDir())
	if err != nil {
		return nil, fmt.Errorf("opening database: %v", err)
	}

	app := app.NewKVStoreApplication(&db)

	pv := privval.LoadFilePV(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	)

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load node's key: %v", err)
	}

	return nm.NewNode(
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		nm.DefaultGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger)
}
