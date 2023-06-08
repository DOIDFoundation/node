package main

import (
	"os"
	"path/filepath"

	"github.com/DOIDFoundation/node/cmd/doidnode/commands"

	"github.com/cometbft/cometbft/libs/cli"
)

func main() {
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "DOID", os.ExpandEnv(filepath.Join("$HOME", ".doidnode")))

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
