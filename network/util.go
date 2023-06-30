package network

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"strings"
)

func (n *Network) buildPeerInfoByAddr(addrs string) peer.AddrInfo {
	///ip4/0.0.0.0/tcp/9000/p2p/QmUyYpeMSqZp4oNMhANdG6sGeckWiGpBnzfCNvP7Pjgbvg
	p2p := strings.TrimSpace(addrs[strings.Index(addrs, "/p2p")+len("/p2p/"):])
	ipTcp := addrs[:strings.Index(addrs, "/p2p/")]
	multiAddr, err := multiaddr.NewMultiaddr(ipTcp)
	if err != nil {
		panic(err)
	}
	m := []multiaddr.Multiaddr{multiAddr}
	id, err := peer.Decode(p2p)
	if err != nil {
		panic(err)
	}
	return peer.AddrInfo{ID: peer.ID(id), Addrs: m}
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
