package config

import (
	"github.com/DOIDFoundation/node/flags"
	"github.com/spf13/viper"
)

type Network = byte

const (
	DevNet  Network = iota
	MainNet         = 1
	TestNet         = 2
)

var (
	NetworkID = DevNet

	// forks, negative value means not forked yet, zero means forked from genesis
	OwnerFork int64 = -1 // fork to store doid by owner
)

func IsTestnet() bool {
	return NetworkID == TestNet
}

func Init() {
	NetworkID = Network(viper.GetInt(flags.NetworkID))
	switch NetworkID {
	case DevNet:
		initDevnetForks()
	case TestNet:
		initTestnetForks()
	}
}
