package network

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type state struct {
	Height uint64
	Td     *big.Int
}

func newState() *state {
	return &state{0, new(big.Int)}
}

func readState(stream network.Stream, peerState *state) error {
	if err := rlp.Decode(stream, peerState); err != nil {
		stream.Reset()
		return err
	}
	return nil
}

func writeStateAndClose(stream network.Stream, peerState *state) error {
	if err := rlp.Encode(stream, peerState); err != nil {
		stream.Reset()
		return err
	}
	if err := stream.CloseWrite(); err != nil {
		return err
	}
	return nil
}

func getPeerState(s peerstore.Peerstore, id peer.ID) *state {
	bz, err := s.Get(id, metaState)
	if err != nil {
		return nil
	}
	peerState := new(state)
	if err := rlp.DecodeBytes(bz.([]byte), peerState); err != nil {
		return nil
	}
	return peerState
}

var peerHasState = make(map[peer.ID]bool)
var peerMutex sync.Mutex

func updatePeerState(peerStore peerstore.Peerstore, peer peer.ID, peerState *state) (updated bool, err error) {
	if oldState := getPeerState(peerStore, peer); oldState != nil && oldState.Td.Cmp(peerState.Td) == 0 {
		return false, nil
	} else if bz, err := rlp.EncodeToBytes(peerState); err != nil {
		return false, err
	} else if err = peerStore.Put(peer, metaState, bz); err != nil {
		return false, err
	}
	peerMutex.Lock()
	defer peerMutex.Unlock()
	peerHasState[peer] = true
	return true, nil
}

func deletePeerState(peerStore peerstore.Peerstore, peer peer.ID) {
	peerStore.RemovePeer(peer)
	peerMutex.Lock()
	defer peerMutex.Unlock()
	delete(peerHasState, peer)
}

func (n *Network) stateHandler(s network.Stream) {
	logger := n.Logger.With("protocol", "state", "peer", s.Conn().RemotePeer())

	logger.Debug("send our state")
	peerState := &state{
		Height: n.blockChain.LatestBlock().Header.Height.Uint64(),
		Td:     new(big.Int).Set(n.blockChain.GetTd()),
	}
	if err := writeStateAndClose(s, peerState); err != nil {
		logger.Debug("failed to send peer state", "err", err, "state", peerState)
	}

	peerState = newState()
	if err := readState(s, peerState); err != nil {
		logger.Debug("failed to read peer state", "err", err)
		s.Reset()
		return
	}

	peer := s.Conn().RemotePeer()
	if updated, err := updatePeerState(n.host.Peerstore(), peer, peerState); updated {
		logger.Debug("peer state updated", "peer", peer, "state", peerState)
		eventPeerState.Send(peer)
	} else if err != nil {
		logger.Error("failed to update peer state", "peer", peer, "err", err)
	}
}
