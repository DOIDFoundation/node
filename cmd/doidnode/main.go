package main

import (
	"github.com/DOIDFoundation/node/cmd/doidnode/commands"
)

func main() {
	if err := commands.RootCmdExecutor().Execute(); err != nil {
		panic(err)
	}
}
