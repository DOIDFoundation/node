package commands

import (
	"os"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger  = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	verbose bool
)

// RootCmd is the root command for doidnode. It is called once in the main
// function.
var RootCmd = &cobra.Command{
	Use:   "doidnode",
	Short: "DOID Network Node",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}

		logger = logger.With("module", "main")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(
		StartCmd,
		VersionCmd,
		cli.NewCompletionCmd(RootCmd, true),
	)
}
