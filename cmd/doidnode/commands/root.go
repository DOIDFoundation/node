package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/cli"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	logger  = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	verbose bool
)

func RootCmdExecutor() cli.Executable {
	// RootCmd is the root command for doidnode. It is called once in the main
	// function.
	RootCmd := &cobra.Command{
		Use:   "doidnode",
		Short: "DOID Network Node",
	}
	RootCmd.AddCommand(
		StartCmd,
		VersionCmd,
		cli.NewCompletionCmd(RootCmd, true),
	)
	// initialize env, home and trace flags
	executor := cli.PrepareBaseCmd(RootCmd, "DOID", os.ExpandEnv(filepath.Join("$HOME", ".doidnode")))
	// initialize network flags
	RootCmd.PersistentFlags().Bool(flags.Testnet, false, "Start from testnet")
	RootCmd.PersistentFlags().Int8(flags.NetworkID, 1, "Explicitly set network id, (For testnets: use --testnet instead)")
	// initialize logging flags
	RootCmd.PersistentFlags().String(flags.Log_Level, "info", "level of logging, can be debug, info, error, none or comma-separated list of module:level pairs with an optional *:level pair (* means all other modules). e.g. 'consensus:debug,mempool:debug,*:error'")
	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		// cmd.Flags() includes flags from this command and all persistent flags from the parent
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}

		homeDir := viper.GetString(flags.Home)
		viper.SetConfigName("config")                         // name of config file (without extension)
		viper.AddConfigPath(".")                              // search current working directory
		viper.AddConfigPath(homeDir)                          // search root directory
		viper.AddConfigPath(filepath.Join(homeDir, "config")) // search root directory /config

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				// ignore not found error, return other errors
				return err
			}
		}

		network := "mainnet"
		networkName := "main network"
		switch {
		case viper.GetBool(flags.Testnet):
			network = "testnet"
			networkName = "test network"
			viper.Set(flags.NetworkID, 2)
		case viper.GetInt(flags.NetworkID) != 1:
			network = fmt.Sprint(viper.GetInt(flags.NetworkID))
			networkName = "private network"
		}

		if homeDir = viper.GetString(flags.Home); homeDir == "." {
			// translate relative path to absolute path
			if path, err := filepath.Abs("."); err == nil {
				viper.Set(flags.Home, path)
			} else {
				return err
			}
		} else if homeDir == cmd.Flag(flags.Home).DefValue && network != "mainnet" {
			// if home flag is not equal to default value, append with network name
			path := filepath.Join(homeDir, network)
			viper.Set(flags.Home, path)
		}

		logger.Info("start "+networkName, "home", viper.GetString(flags.Home),
			"networkid", viper.GetInt(flags.NetworkID),
			"config", viper.ConfigFileUsed(),
			"loglevel", viper.GetString(flags.Log_Level))

		if viper.GetBool(flags.Trace) {
			logger = log.NewTracingLogger(logger)
		}

		logger, err = cmtflags.ParseLogLevel(viper.GetString(flags.Log_Level), logger.With("module", "main"), cmd.Flag(flags.Log_Level).DefValue)
		return err
	}
	cmd, _, err := RootCmd.Find(os.Args[1:])
	// execute start cmd if no cmd is given
	if err == nil && cmd.Use == RootCmd.Use && cmd.Flags().Parse(os.Args[1:]) != pflag.ErrHelp {
		args := append([]string{StartCmd.Use}, os.Args[1:]...)
		RootCmd.SetArgs(args)
	}
	return executor
}
