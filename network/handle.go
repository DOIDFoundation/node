package network

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func (n *Network) handleStream(stream network.Stream) {
	data, err := ioutil.ReadAll(stream)
	if err != nil {
		log.Panic(err)
	}
	cmd, content := splitMessage(data)
	n.Logger.Debug("cmd", "data", len(data), "content", len(content), "from", stream.Conn().RemotePeer())
	switch command(cmd) {
	case cMyError:
		go n.handleMyError(content)
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
	stream, err := n.host.NewStream(ctx, peer.ID, protocol.ID(ProtocolID))
	if err != nil {
		n.Logger.Error("Stream open failed", "err", err, "peer", peer)
	} else {
		cmd, _ := splitMessage(data)

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
	ss := "network:" + n.host.ID().String() + " exist"
	m := myerror{ss, n.host.ID().String()}
	data := jointMessage(cMyError, m.serialize())
	for _, v := range peerPool {
		peerAddr := peer.AddrInfo{ID: peer.ID(v.ID), Addrs: v.Addrs}
		n.SendMessage(peerAddr, data)
	}
}
