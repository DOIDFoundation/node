package network

import (
	"bufio"
	"fmt"
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"io/ioutil"
	"log"
	"math/big"
	"sync"
)

func (n *Network) handleStream(stream network.Stream) {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		log.Panic(err)
	}
	cmd, content := n.splitMessage(data)
	n.Logger.Info("cmd", "data", len(data), "content", len(content))
	switch command(cmd) {
	case cVersion:
		go n.handleVersion(content)
	case cGetBlock:
		go n.handleGetBlock(content)
	case cBlock:
		go n.handleBlock(content)
	case cMyError:
		go n.handleMyError(content)
	}
}

func (n *Network) handleVersion(content []byte) {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	v := new(version)
	v.deserialize(content)

	n.Logger.Info("handle version", "peer", v.AddrFrom)
	if n.networkHeight.Cmp(big.NewInt(v.Height)) < 0 {
		n.networkHeight.Set(big.NewInt(v.Height))
		if n.blockChain.LatestBlock().Header.Height.Cmp(big.NewInt(v.Height)) < 0 {
			n.Logger.Info("we are behind, start sync", "ours", n.blockChain.LatestBlock().Header.Height, "network", v.Height)

			gh := getBlock{Height: n.blockChain.LatestBlock().Header.Height.Int64() + 1, AddrFrom: localAddr}
			data := n.jointMessage(cGetBlock, gh.serialize())
			n.SendMessage(n.buildPeerInfoByAddr(v.AddrFrom), data)
		}
	}
}

func (n *Network) handleGetBlock(content []byte) {
	v := getBlock{}
	v.deserialize(content)

	n.Logger.Info("handle getblock", "height", v.Height)
	if big.NewInt(v.Height).Cmp(n.blockChain.LatestBlock().Header.Height) <= 0 {
		block := n.blockChain.BlockByHeight(uint64(v.Height))

		b, err := rlp.EncodeToBytes(block)
		n.Logger.Info("encode block for broadcasting", "size", len(b), "height", v.Height)
		if err != nil {
			n.Logger.Info("failed to encode block for broadcasting", "err", err)
		}
		fmt.Printf("send block size: %d\n", len(b))
		gb := Block{BlockHash: b, AddrFrom: localAddr}
		data := n.jointMessage(cBlock, gb.serialize())
		n.SendMessage(n.buildPeerInfoByAddr(v.AddrFrom), data)
	}
}

func (n *Network) handleBlock(content []byte) {
	v := Block{}
	v.deserialize(content)

	block := new(types.Block)
	err := rlp.DecodeBytes(v.BlockHash, block)
	if err != nil {
		n.Logger.Info("failed to decode received block", "Peer", v.AddrFrom, "err", err)
		return
	}
	if n.blockChain.LatestBlock().Header.Height.Int64() < block.Header.Height.Int64() {
		n.Logger.Info("apply block", "Peer", v.AddrFrom, "block height", block.Header.Height)
		err = n.blockChain.InsertBlocks([]*types.Block{block})
		if err != nil {
			n.Logger.Info("failed to apply block", "Peer", v.AddrFrom, "err", err)

		}

		if n.blockChain.LatestBlock().Header.Height.Cmp(n.networkHeight) < 0 {
			n.Logger.Info("we are behind, start sync", "ours", n.blockChain.LatestBlock().Header.Height, "network", n.networkHeight)

			gh := getBlock{Height: n.blockChain.LatestBlock().Header.Height.Int64() + 1, AddrFrom: localAddr}
			data := n.jointMessage(cGetBlock, gh.serialize())
			n.SendMessage(n.buildPeerInfoByAddr(v.AddrFrom), data)
		}
	} else {
		return
	}
}

func (n *Network) handleMyError(content []byte) {
	e := myerror{}
	e.deserialize(content)
	n.Logger.Info(e.Error)
	peer := e.Addrfrom
	delete(n.peerPool, fmt.Sprint(peer))
}

func (n *Network) SendMessage(peer peer.AddrInfo, data []byte) {
	if err := n.h.Connect(ctx, peer); err != nil {
		n.Logger.Error("Connection failed:", err)
	}

	stream, err := n.h.NewStream(ctx, peer.ID, protocol.ID(ProtocolID))
	if err != nil {
		n.Logger.Info("Stream open failed", err)
	} else {
		cmd, _ := n.splitMessage(data)

		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		_, err := rw.Write(data)
		if err != nil {
			n.Logger.Info(err.Error())
		}
		err = rw.Flush()
		if err != nil {
			n.Logger.Info(err.Error())
		}
		err = stream.Close()
		if err != nil {
			n.Logger.Info(err.Error())
		}
		n.Logger.Info("stream", "sendCmd", cmd, "toPeer", peer)
	}
}

func (n *Network) SendSignOutToPeers() {
	ss := "network:" + localAddr + " exist"
	m := myerror{ss, localAddr}
	data := n.jointMessage(cMyError, m.serialize())
	for _, v := range peerPool {
		peerAddr := peer.AddrInfo{ID: peer.ID(v.ID), Addrs: v.Addrs}
		n.SendMessage(peerAddr, data)
	}
}
