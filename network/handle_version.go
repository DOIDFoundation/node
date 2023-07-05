package network

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type peerState struct {
	//Version  byte
	Height uint64
	Td     *big.Int
	ID     string
}

func (v peerState) serialize() []byte {
	bz, err := rlp.EncodeToBytes(v)
	if err != nil {
		panic(err)
	}
	return bz
}

func (v *peerState) deserialize(d []byte) {
	err := rlp.DecodeBytes(d, v)
	if err != nil {
		panic(err)
	}
}

type state struct {
	Height uint64
	Td     *big.Int
}

const (
	timeoutState = time.Second * 60
)

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

func (n *Network) updatePeerState(peer peer.ID, peerState *state) {
	n.Logger.Info("update peer state", "state", peerState, "peer", peer)
	bz, err := rlp.EncodeToBytes(peerState)
	if err == nil {
		err = n.host.Peerstore().Put(peer, metaState, bz)
	}
	if err != nil {
		n.Logger.Error("failed to update peer state", "err", err, "peer", peer)
	}
}

func (n *Network) updatePeerStateAndSync(peer peer.ID, peerState *state) {
	n.updatePeerState(peer, peerState)
	td := n.blockChain.GetTd()
	if td.Cmp(peerState.Td) < 0 {
		n.Logger.Info("we are behind, start sync", "ourHeight", n.blockChain.LatestBlock().Header.Height, "ourTD", td, "networkHeight", peerState.Height, "networkTD", peerState.Td)
		n.startSync()
	}
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

	peerState = &state{0, new(big.Int)}
	if err := readState(s, peerState); err != nil {
		logger.Debug("failed to read peer state", "err", err)
		s.Reset()
		return
	}
	n.updatePeerStateAndSync(s.Conn().RemotePeer(), peerState)
}
