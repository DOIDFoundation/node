package commands

import (
	"fmt"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/node"
	"github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"
)

// expose configuration options for starting a node.
func init() {
	StartCmd.Flags().String(flags.DB_Engine, "goleveldb", "Backing database implementation to use ('memdb' or 'goleveldb')")
	StartCmd.Flags().BoolP(flags.Mine_Enabled, "m", false, "Enable mining")
	StartCmd.Flags().Uint(flags.Mine_Threads, 0, "Number of threads to start mining, 0 indicates number of logical CPUs")
	StartCmd.Flags().String(flags.RPC_Addr, "127.0.0.1:26657", "RPC listen address")
	StartCmd.Flags().String(flags.P2P_Addr, "/ip4/127.0.0.1/tcp/26667", "Libp2p listen address")
	StartCmd.Flags().StringP(flags.P2P_Rendezvous, "r", "doidnode", "Libp2p rendezvous string used for peer discovery")
	StartCmd.Flags().String(flags.P2P_Key, "", "Private key to generate libp2p peer identity")
	StartCmd.Flags().String(flags.P2P_KeyFile, "p2p.key", "Private key file to generate libp2p peer identity")
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
