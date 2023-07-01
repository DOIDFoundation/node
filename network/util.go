package network

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

func (n *Network) buildPeerInfoByAddr(addrs string) peer.AddrInfo {
	id, err := peer.Decode(addrs)
	if err != nil {
		n.Logger.Error("failed to decode peer id", "err", err, "id", addrs)
		return peer.AddrInfo{}
	}
	return n.host.Peerstore().PeerInfo(id)
}

func (n *Network) jointMessage(cmd command, content []byte) []byte {
	b := make([]byte, prefixCMDLength)
	for i, v := range []byte(cmd) {
		b[i] = v
	}
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

func (n *Network) splitMessage(message []byte) (cmd string, content []byte) {
	cmdBytes := message[:prefixCMDLength]
	newCMDBytes := make([]byte, 0)
	for _, v := range cmdBytes {
		if v != byte(0) {
			newCMDBytes = append(newCMDBytes, v)
		}
	}
	cmd = string(newCMDBytes)
	content = message[prefixCMDLength:]
	return
}
