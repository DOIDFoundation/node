package commands

import (
	"os"

	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/cli"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
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
		viper.AddConfigPath(".")
		if viper.GetBool(flags.Trace) {
			logger = log.NewTracingLogger(logger)
		}

		logger, err = cmtflags.ParseLogLevel(viper.GetString(flags.Log_Level), logger.With("module", "main"), cmd.Flag(flags.Log_Level).DefValue)
		return err
	},
}

func init() {
	RootCmd.PersistentFlags().String(flags.Log_Level, "info", "level of logging, can be debug, info, error, none or comma-separated list of module:level pairs with an optional *:level pair (* means all other modules). e.g. 'consensus:debug,mempool:debug,*:error'")
	RootCmd.AddCommand(
		StartCmd,
		VersionCmd,
		cli.NewCompletionCmd(RootCmd, true),
	)
}
