package commands

import (
	"fmt"

	"github.com/DOIDFoundation/node/network"
	"github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"
)

// addFlags exposes configuration options for starting a network.
func addFlags(cmd *cobra.Command) {
}

// StartCmd is the command that allows the CLI to start a network.
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"network", "run"},
	Short:   "Run the CometBFT network",
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := network.NewNode(logger)
		if err != nil {
			return fmt.Errorf("failed to create network: %w", err)
		}

		if err := n.Start(); err != nil {
			return fmt.Errorf("failed to start network: %w", err)
		}

		logger.Info("started network")

		// Stop upon receiving SIGTERM or CTRL-C.
		os.TrapSignal(logger, func() {
			if n.IsRunning() {
				if err := n.Stop(); err != nil {
					logger.Error("unable to stop the network", "error", err)
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
