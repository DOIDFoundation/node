package rpc

import (
	"net"
	"net/http"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	ethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/viper"
)

var APIs []rpc.API // List of APIs currently provided by the network

type RPC struct {
	service.BaseService
	config *Config
	server *http.Server
}

func NewRPC(logger log.Logger) *RPC {
	rpc := &RPC{config: &DefaultConfig}
	rpc.BaseService = *service.NewBaseService(logger, "RPC", rpc)
	return rpc
}

func (r *RPC) OnStart() error {
	ethlog.Root().SetHandler(
		ethlog.FuncHandler(func(record *ethlog.Record) error {
			fn := r.Logger.Info
			switch record.Lvl {
			case ethlog.LvlTrace, ethlog.LvlDebug:
				fn = r.Logger.Debug
			case ethlog.LvlError, ethlog.LvlCrit:
				fn = r.Logger.Error
			}
			fn(record.Msg, record.Ctx...)
			return nil
		}))

	listenAddr := viper.GetString("rpc.addr")
	// Initialize the server.
	rpcServer := rpc.NewServer()

	// Register RPC services.
	for _, rpcAPI := range APIs {
		if err := rpcServer.RegisterName(rpcAPI.Namespace, rpcAPI.Service); err != nil {
			return err
		}
	}

	r.server = &http.Server{
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
	go r.server.Serve(listener)
	return nil
}

func (r *RPC) OnStop() {
	r.server.Close()
}

func RegisterName(name string, receiver interface{}) {
	APIs = append(APIs, rpc.API{Namespace: name, Service: receiver})
}
