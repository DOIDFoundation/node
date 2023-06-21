package mempool

import (
	"github.com/DOIDFoundation/node/rpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type API struct {
	pool *Mempool
}

// Status returns the number of pending and queued transaction in the pool.
func (s *API) Status() map[string]hexutil.Uint {
	local, remote := s.pool.Stats()
	return map[string]hexutil.Uint{
		"local":  hexutil.Uint(local),
		"remote": hexutil.Uint(remote),
	}
}

func (pool *Mempool) RegisterAPI() {
	rpc.RegisterName("txpool", &API{pool: pool})
}
