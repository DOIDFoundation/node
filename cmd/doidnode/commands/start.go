package commands

import (
	"fmt"

	"github.com/DOIDFoundation/node/node"
	"github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"
)

// addFlags exposes configuration options for starting a node.
func addFlags(cmd *cobra.Command) {
}

// StartCmd is the command that allows the CLI to start a node.
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"node", "run"},
	Short:   "Run the CometBFT node",
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

func init() {
	addFlags(StartCmd)
}
