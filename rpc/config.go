package rpc

import "github.com/ethereum/go-ethereum/rpc"

// Defines the configuration options for the RPC server
type Config struct {
	// TCP or UNIX socket address for the RPC server to listen on
	ListenAddress string `mapstructure:"laddr"`
	// HTTPTimeouts allows for customization of the timeout values used by the HTTP RPC
	// interface.
	HTTPTimeouts rpc.HTTPTimeouts
}

// DefaultConfig returns a default configuration for the RPC server
var DefaultConfig = Config{
	ListenAddress: "127.0.0.1:26657",
	HTTPTimeouts:  rpc.DefaultHTTPTimeouts,
}
