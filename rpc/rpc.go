package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/DOIDFoundation/node/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	ethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/viper"
)

var APIs []rpc.API // List of APIs currently provided by the network

type RPC struct {
	service.BaseService
	config       *Config
	rpcServer    *rpc.Server
	http         *http.Server
	httpListener net.Listener
	ws           *http.Server
	wsListener   net.Listener
}

func NewRPC(logger log.Logger) *RPC {
	rpc := &RPC{config: &DefaultConfig}
	rpc.BaseService = *service.NewBaseService(logger.With("module", "rpc"), "RPC", rpc)
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

	// Initialize the server.
	r.rpcServer = rpc.NewServer()

	// Register RPC services.
	for _, rpcAPI := range APIs {
		if err := r.rpcServer.RegisterName(rpcAPI.Namespace, rpcAPI.Service); err != nil {
			return err
		}
	}

	if viper.GetBool(flags.RPC_Http) {
		if err := r.enableHttp(r.rpcServer); err != nil {
			r.Logger.Error("failed to enable HTTP RPC", "err", err)
		}
	}
	if viper.GetBool(flags.RPC_Ws) {
		if err := r.enableWs(r.rpcServer.WebsocketHandler(viper.GetStringSlice(flags.RPC_WsOrigins))); err != nil {
			r.Logger.Error("failed to enable WebSocket RPC", "err", err)
		}
	}
	return nil
}

func (r *RPC) OnStop() {
	if r.http != nil {
		r.http.Shutdown(context.Background())
		r.httpListener.Close()
	}
	if r.ws != nil {
		r.ws.Shutdown(context.Background())
		r.wsListener.Close()
	}
}

func (r *RPC) enableHttp(rpcServer http.Handler) error {
	listenAddr := viper.GetString(flags.RPC_HttpAddr)

	r.Logger.Debug("try listening", "listenAddr", listenAddr)
	// Start the server.
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	r.httpListener = listener
	r.http = &http.Server{
		Handler:           rpcServer,
		ReadTimeout:       r.config.HTTPTimeouts.ReadTimeout,
		ReadHeaderTimeout: r.config.HTTPTimeouts.ReadHeaderTimeout,
		WriteTimeout:      r.config.HTTPTimeouts.WriteTimeout,
		IdleTimeout:       r.config.HTTPTimeouts.IdleTimeout,
	}
	go r.http.Serve(listener)
	r.Logger.Info("HTTP server started", "endpoint", listener.Addr())
	return nil
}

func (r *RPC) enableWs(rpcServer http.Handler) error {
	listenAddr := viper.GetString(flags.RPC_WsAddr)

	r.Logger.Debug("try listening", "listenAddr", listenAddr)
	// Start the server.
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	r.wsListener = listener
	r.ws = &http.Server{
		Handler:           rpcServer,
		ReadTimeout:       r.config.HTTPTimeouts.ReadTimeout,
		ReadHeaderTimeout: r.config.HTTPTimeouts.ReadHeaderTimeout,
		WriteTimeout:      r.config.HTTPTimeouts.WriteTimeout,
		IdleTimeout:       r.config.HTTPTimeouts.IdleTimeout,
	}
	go r.ws.Serve(listener)
	r.Logger.Info("WebSocket enabled", "url", fmt.Sprintf("ws://%v", listener.Addr()))
	return nil
}

func RegisterName(name string, receiver interface{}) {
	APIs = append(APIs, rpc.API{Namespace: name, Service: receiver})
}
