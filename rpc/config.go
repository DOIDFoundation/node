package rpc

import "github.com/ethereum/go-ethereum/rpc"

// Defines the configuration options for the RPC server
type Config struct {
	// HTTPTimeouts allows for customization of the timeout values used by the HTTP RPC
	// interface.
	HTTPTimeouts rpc.HTTPTimeouts
}

// DefaultConfig returns a default configuration for the RPC server
var DefaultConfig = Config{
	HTTPTimeouts: rpc.DefaultHTTPTimeouts,
}
