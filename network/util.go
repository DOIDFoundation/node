package network

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

func peerIDFromString(s string) peer.ID {
	id, err := peer.Decode(s)
	if err != nil {
		return ""
	}
	return id
}

func (n *Network) peerInfoByID(s string) peer.AddrInfo {
	return n.host.Peerstore().PeerInfo(peerIDFromString(s))
}

func jointMessage(cmd command, content []byte) []byte {
	b := make([]byte, prefixCMDLength)
	for i, v := range []byte(cmd) {
		b[i] = v
	}
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

func splitMessage(message []byte) (cmd string, content []byte) {
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
