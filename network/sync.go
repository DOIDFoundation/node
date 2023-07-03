package network

import (
	"errors"
	"fmt"
	"math/big"

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
}

func newSyncService(logger log.Logger, id peer.ID, h host.Host, chain *core.BlockChain) *syncService {
	s := &syncService{id: id, host: h, chain: chain, hc: chain.NewHeaderChain()}
	s.BaseService = *service.NewBaseService(logger.With("service", "sync", "peer", id), "Sync", s)
	return s
}

func (s *syncService) OnStart() error {
	go s.sync()
	return nil
}

func (s *syncService) OnStop() {
}

func (s *syncService) dropPeer() {
	s.Logger.Info("drop peer")
	s.host.Network().ClosePeer(s.id)
	s.host.Peerstore().RemovePeer(s.id)
}

func (s *syncService) sync() {
	s.doSync()
	events.SyncFinished.Send(struct{}{})
}

func (s *syncService) doSync() {
	stream, err := s.host.NewStream(ctx, s.id, protocol.ID(ProtocolGetBlock))
	if err != nil {
		s.Logger.Error("failed to create stream", "err", err)
		s.dropPeer()
		return
	}
	localHeight := s.chain.LatestBlock().Header.Height
	v, err := s.host.Peerstore().Get(s.id, metaVersion)
	if err != nil {
		s.Logger.Error("failed to get peer version", "err", err)
		s.dropPeer()
		return
	}
	remoteHeight := v.(peerState).Height
	ancestorHeight := uint64(0)
	// find which block we can start sync from
	if remoteHeight > localHeight.Uint64() {
		// try next block after local height
		check := localHeight.Uint64() + 1
		blocks, err := s.getBlocks(stream, check, 1)
		if err != nil {
			s.Logger.Info("failed to get block", "err", err)
			s.dropPeer()
			return
		}
		block := blocks[0]
		if err := s.chain.ApplyBlock(block); err != nil {
			// failed to apply block, check if we can find ancestor
			if errors.Is(err, types.ErrNotContiguous) {
				ancestorHeight = s.findAncestor(stream, localHeight.Uint64())
			} else {
				s.Logger.Info("failed to apply block", "err", err, "header", block.Header, "hash", block.Hash())
				s.dropPeer()
				return
			}
		} else if check >= remoteHeight {
			// only one block behind, no more to sync
			return
		} else {
			// current head is ancestor
			ancestorHeight = check
		}
	} else {
		ancestorHeight = s.findAncestor(stream, remoteHeight)
	}
	if ancestorHeight == 0 {
		s.Logger.Info("failed to find ancestor")
		s.dropPeer()
		return
	}
	// now start batch sync after ancestor
	start := ancestorHeight + 1
	s.Logger.Info("start sync", "from", start, "to", remoteHeight)
	for start <= remoteHeight {
		count := remoteHeight - start + 1
		if count > 16 {
			count = 16
		}
		blocks, err := s.getBlocks(stream, start, count)
		if err != nil {
			s.Logger.Info("can not get blocks", "from", start, "count", count, "err", err)
			s.dropPeer()
			return
		}
		// add to header chain until we have enough blocks
		if err := s.hc.AppendBlocks(blocks); err != nil {
			s.Logger.Info("can not append blocks", "from", start, "count", count, "err", err)
			s.dropPeer()
			return
		}
		start += 16
	}
	// apply blocks synced or drop peer
	if err := s.chain.ApplyHeaderChain(s.hc); err != nil {
		s.Logger.Info("can not apply header chain", "from", start, "err", err)
		s.dropPeer()
	}
}

func (s *syncService) getBlocks(stream network.Stream, height uint64, count uint64) ([]*types.Block, error) {
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
	return *blocks, nil
}

func (s *syncService) findAncestor(stream network.Stream, height uint64) uint64 {
	s.Logger.Info("need to find ancestor", "till", height)
	// try search 12x16 blocks backwards
	check := uint64(1)
	for count := uint64(1); count <= 12; count-- {
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
			if err := s.hc.AppendBlocks([]*types.Block{b}); err != nil {
				s.Logger.Debug("failed to append block", "err", err, "height", check)
				continue
			}
			s.Logger.Info("ready to start sync", "from", check)
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
				if err := s.hc.AppendBlocks([]*types.Block{b}); err != nil {
					s.Logger.Debug("failed to append block", "err", err, "height", check)
					end = check
					continue
				}
				return check
			}
			start = check
		} else {
			end = check
		}
	}
	return start
}
