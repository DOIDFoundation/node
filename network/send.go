package network

import (
	"bufio"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Send struct {
	node *Node
}

func (s Send) SendSignOutToPeers() {
	ss := "network:" + localAddr + " exist"
	m := myerror{ss, localAddr}
	data := jointMessage(cMyError, m.serialize())
	for _, v := range peerPool {
		peerAddr := peer.AddrInfo{ID: peer.ID(v.ID), Addrs: v.Addrs}
		s.SendMessage(peerAddr, data)
	}
}

func (s Send) SendTestToPeers() {
	ss := "network:" + localAddr + " TEST"
	m := myerror{ss, localAddr}
	data := jointMessage(cMyTest, m.serialize())

	for _, v := range peerPool {
		ss, _ := v.MarshalJSON()
		logger.Info(string(ss))

		s.SendMessage(v, data)
	}
}

func (s Send) SendMessage(peer peer.AddrInfo, data []byte) {
	if err := s.node.localHost.Connect(ctx, peer); err != nil {
		logger.Error("Connection failed:", err)
	}

	stream, err := s.node.localHost.NewStream(ctx, peer.ID, protocol.ID(ProtocolID))
	if err != nil {
		logger.Info("Stream open failed", err)
	} else {
		cmd, _ := splitMessage(data)

		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		_, err := rw.Write(data)
		if err != nil {
			logger.Debug(err.Error())
		}
		err = rw.Flush()
		if err != nil {
			logger.Debug(err.Error())
		}
		err = stream.Close()
		if err != nil {
			logger.Debug(err.Error())
		}
		logger.Debug("send cmd:%s to peer:%v", cmd, peer)
	}
}
