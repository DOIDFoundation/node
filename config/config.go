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

	OwnerFork ForkNumber // fork to store doid by owner
	UncleFork ForkNumber // fork to enable uncle in header
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
