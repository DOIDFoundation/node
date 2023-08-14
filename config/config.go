package config

import (
	"github.com/DOIDFoundation/node/flags"
	"github.com/spf13/viper"
)

var NetworkID byte

func Init() {
	NetworkID = byte(viper.GetInt(flags.NetworkID))
}
