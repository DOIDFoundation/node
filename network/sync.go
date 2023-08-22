package network

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type syncService struct {
	service.BaseService
	id    peer.ID
	host  host.Host
	chain *core.BlockChain
	hc    *core.HeaderChain
	abort chan struct{}
	wg    sync.WaitGroup
}

func newSyncService(logger log.Logger, id peer.ID, h host.Host, chain *core.BlockChain) *syncService {
	s := &syncService{id: id, host: h, chain: chain, hc: chain.NewHeaderChain()}
	s.BaseService = *service.NewBaseService(logger.With("module", "network", "service", "sync", "peer", id), "Sync", s)
	return s
}

func (s *syncService) OnStart() error {
	s.wg.Add(1)
	s.abort = make(chan struct{})
	go s.sync()
	return nil
}

func (s *syncService) OnStop() {
	close(s.abort)
	s.wg.Wait()
}

func (s *syncService) dropPeer() {
	s.Logger.Info("drop peer")
	s.host.Network().ClosePeer(s.id)
	deletePeerState(s.host.Peerstore(), s.id)
}

func (s *syncService) sync() {
	defer s.wg.Done()
	s.doSync()
	events.SyncFinished.Send()
}

func (s *syncService) doSync() bool {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15) // @todo add a config/flag for this timeout
	defer cancel()
	// @todo remove testnet legacy protocol id.
	stream, err := s.host.NewStream(ctx, s.id, protocol.ID(ProtocolGetBlocks), protocol.ID(strings.Replace(ProtocolGetBlocks, "/doid/2/", "/doid/", 1)))
	if err != nil {
		s.Logger.Error("failed to create stream", "err", err)
		s.dropPeer()
		return false
	}
	defer stream.Close()
	v := getPeerState(s.host.Peerstore(), s.id)
	if v == nil {
		s.Logger.Error("failed to get peer version")
		s.dropPeer()
		return false
	}
	remoteHeight := v.Height
	// find which block we can start sync from
	ancestorHeight := s.findAncestor(stream, remoteHeight)
	if ancestorHeight == 0 {
		s.Logger.Info("failed to find ancestor")
		s.dropPeer()
		return false
	}
	// now start batch sync after ancestor
	start := ancestorHeight + 1
	s.Logger.Info("start sync", "from", start, "to", remoteHeight)
	for start <= remoteHeight {
		select {
		case <-s.abort:
			s.Logger.Info("abort sync")
			return false
		default:
		}
		count := remoteHeight - start + 1
		if count > 16 {
			count = 16
		}
		blocks, err := s.getBlocks(stream, start, count)
		if err != nil {
			s.Logger.Info("can not get blocks", "from", start, "count", count, "err", err)
			s.dropPeer()
			return false
		}
		// add to header chain until we have enough blocks
		if err := s.hc.AppendBlocks(blocks); err != nil {
			s.Logger.Info("can not append blocks", "from", start, "count", count, "err", err)
			s.dropPeer()
			return false
		}
		start += 16
	}
	// apply blocks synced or drop peer
	if err := s.chain.ApplyHeaderChain(s.hc); err != nil {
		s.Logger.Info("can not apply header chain", "from", start, "err", err)
		s.dropPeer()
		return false
	}
	return true
}

func (s *syncService) getBlocks(stream network.Stream, height uint64, count uint64) ([]*types.Block, error) {
	stream.SetDeadline(time.Now().Add(time.Second * 15)) // @todo add config/flag for this timeout
	if err := rlp.Encode(stream, requestBlocks{new(big.Int).SetUint64(height), new(big.Int).SetUint64(count)}); err != nil {
		return nil, err
	}
	blocks := new([]*types.Block)
	if err := rlp.Decode(stream, blocks); err != nil {
		return nil, err
	}
	if len(*blocks) != int(count) {
		return nil, fmt.Errorf("want %v blocks, got %v", count, len(*blocks))
	}
	s.Logger.Info("got blocks", "from", height, "count", count, "gap", time.Since(time.Unix(int64((*blocks)[len(*blocks)-1].Header.Time), 0)))
	return *blocks, nil
}

func (s *syncService) findAncestor(stream network.Stream, height uint64) uint64 {
	localHeight := s.chain.LatestBlock().Header.Height
	if localHeight.Uint64() == 1 {
		s.Logger.Info("check if we need to start from genesis")
		check := uint64(2)
		blocks, err := s.getBlocks(stream, check, 1)
		if err != nil {
			s.Logger.Info("can not get block", "height", check)
			return 0
		}
		b := blocks[0]
		if !s.hc.CanStartFrom(check, b.Hash()) {
			// we don't have this block, apply and set as ancestor
			if err := s.hc.AppendBlocks([]*types.Block{b}); err != nil {
				s.Logger.Debug("failed to append block", "err", err, "height", check)
				s.dropPeer()
				return 0
			}
			return check
		}
	}
	s.Logger.Info("need to find ancestor", "till", height)
	// try search 12x16 blocks backwards
	check := uint64(1)
	for count := uint64(1); count <= 12; count++ {
		skip := count * 16
		if height > skip {
			check = height - skip
		} else {
			s.Logger.Info("not enough blocks, sync from genesis")
			return 1
		}
		s.Logger.Debug("check block", "height", check)
		blocks, err := s.getBlocks(stream, check, 1)
		if err != nil {
			s.Logger.Info("can not get block", "height", check)
			return 0
		}
		b := blocks[0]
		if s.hc.CanStartFrom(check, b.Hash()) {
			return check
		}
	}

	// try binary search if still not found
	start, end := uint64(1), check
	s.Logger.Debug("check deeper", "till", end)
	for start+16 < end {
		// Split our chain interval in two, and request the hash to cross check
		check = (start + end) / 2
		blocks, err := s.getBlocks(stream, check, 1)
		if err != nil {
			s.Logger.Info("can not get block", "height", check)
			return 0
		}
		b := blocks[0]
		if s.hc.CanStartFrom(check, b.Hash()) {
			if end-check < 16 {
				return check
			}
			start = check
		} else {
			end = check
		}
	}
	return start
}
