package commands

import (
	"fmt"

	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/node"
	"github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addFlags exposes configuration options for starting a node.
func addFlags(cmd *cobra.Command) {
	cmd.Flags().String(flags.RPC_Addr, "127.0.0.1:26657", "rpc listen address")
	cmd.Flags().String(flags.P2P_Addr, "/ip4/127.0.0.1/tcp/26667", "p2p listen address")
	cmd.Flags().String(flags.DB_Engine, "goleveldb", "Backing database implementation to use ('memdb' or 'goleveldb')")
	cmd.Flags().StringP("rendezvous", "r", "", "rendezvous")
	viper.BindPFlags(cmd.Flags())
}

// StartCmd is the command that allows the CLI to start a node.
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"node", "run"},
	Short:   "Run the DOID node",
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := node.NewNode(logger)
		if err != nil {
			return fmt.Errorf("failed to create node: %w", err)
		}

		backend, err := doid.New(n)
		backend.StartMining()
		if err != nil {
			return fmt.Errorf("failed to create backend: %w", err)
		}

		if err := n.Start(); err != nil {
			return fmt.Errorf("failed to start node: %w", err)
		}

		logger.Info("started node")

		// Stop upon receiving SIGTERM or CTRL-C.
		os.TrapSignal(logger, func() {
			if n.IsRunning() {
				if err := n.Stop(); err != nil {
					logger.Error("unable to stop the node", "error", err)
				}
			}
		})

		// Run forever.
		select {}
	},
}

func init() {
	addFlags(StartCmd)
}
