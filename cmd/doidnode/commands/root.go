package commands

import (
	"github.com/cometbft/cometbft/cmd/cometbft/commands"
	"github.com/cometbft/cometbft/libs/cli"
	"github.com/spf13/cobra"
)

// NewRootCmd creates a new root command for doidnode. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	rootCmd := commands.RootCmd
	rootCmd.Use = "doidnode"
	rootCmd.Short = "DOID Network Node"

	initRootCmd(rootCmd)

	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command) {
	initCmd := commands.InitFilesCmd
	initCmd.Short = "Initialize DOID Node"

	startCmd := commands.NewRunNodeCmd(NewNode)
	startCmd.Short = "Run the DOID Node"

	rootCmd.AddCommand(
		initCmd,
		startCmd,
		cli.NewCompletionCmd(rootCmd, true))
}
