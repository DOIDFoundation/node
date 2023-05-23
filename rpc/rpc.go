package rpc

import (
	"net"
	"net/http"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rpc"
)

var APIs []rpc.API // List of APIs currently provided by the node

type RPC struct {
	service.BaseService
	config *Config
}

func NewRPC(logger log.Logger) *RPC {
	rpc := &RPC{config: &DefaultConfig}
	rpc.BaseService = *service.NewBaseService(logger, "RPC", rpc)
	return rpc
}

func (r *RPC) OnStart() error {
	listenAddr := r.config.ListenAddress
	// Initialize the server.
	rpcServer := rpc.NewServer()

	// Register RPC services.
	for _, rpcAPI := range APIs {
		if err := rpcServer.RegisterName(rpcAPI.Namespace, rpcAPI.Service); err != nil {
			return err
		}
	}

	server := &http.Server{
		Handler:           rpcServer,
		ReadTimeout:       r.config.HTTPTimeouts.ReadTimeout,
		ReadHeaderTimeout: r.config.HTTPTimeouts.ReadHeaderTimeout,
		WriteTimeout:      r.config.HTTPTimeouts.WriteTimeout,
		IdleTimeout:       r.config.HTTPTimeouts.IdleTimeout,
	}

	r.Logger.Debug("try listening", "listenAddr", listenAddr)
	// Start the server.
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	r.Logger.Info("listening", "listenAddr", listenAddr)
	go server.Serve(listener)
	return nil
}

func (r *RPC) OnStop() {
}

func RegisterName(name string, receiver interface{}) {
	APIs = append(APIs, rpc.API{Namespace: name, Service: receiver})
}
