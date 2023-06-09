package network

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/network"
)

type requestBlocks struct {
	From  *big.Int
	Count *big.Int
}

const (
	timeoutGetBlock = time.Second * 60
)

func (n *Network) getBlocksHandler(s network.Stream) {
	logger := n.Logger.With("protocol", "getblocks", "peer", s.Conn().RemotePeer())
	logger.Debug("stream established")

	errCh := make(chan error, 1)
	defer close(errCh)
	timer := time.NewTimer(timeoutGetBlock)
	defer timer.Stop()

	go func() {
		select {
		case <-timer.C:
			logger.Debug("get timeout")
		case err, ok := <-errCh:
			if ok {
				if errors.Is(err, io.EOF) {
					logger.Debug("got eof")
				} else {
					logger.Error("failed", "err", err)
				}
			} else {
				logger.Error("failed without error")
			}
		}
		s.Close()
	}()

	for {
		req := new(requestBlocks)
		if err := rlp.Decode(s, req); err != nil {
			errCh <- err
			return
		}
		logger.Debug("want blocks", "request", req)

		blocks := []*types.Block{}
		for i := uint64(0); i < req.Count.Uint64(); i++ {
			block := n.blockChain.BlockByHeight(req.From.Uint64() + i)
			if block == nil {
				errCh <- fmt.Errorf("block %v not found", req.From.Uint64()+i)
				return
			}
			blocks = append(blocks, block)
		}

		if err := rlp.Encode(s, blocks); err != nil {
			errCh <- err
			return
		}

		timer.Reset(timeoutGetBlock)
	}
}
