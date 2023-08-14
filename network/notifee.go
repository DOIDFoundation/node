package network

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
)

// called when network starts listening on an addr
func (n *Network) Listen(network.Network, multiaddr.Multiaddr) {

}

// called when network stops listening on an addr
func (n *Network) ListenClose(network.Network, multiaddr.Multiaddr) {
}

// called when a connection opened
func (n *Network) Connected(network network.Network, c network.Conn) {
	// n.Logger.Debug("connected", "peer", c.RemotePeer())
}

// called when a connection closed
func (n *Network) Disconnected(_ network.Network, c network.Conn) {
	// n.Logger.Debug("disconnected", "peer", c.RemotePeer())
	deletePeerState(n.host.Peerstore(), c.RemotePeer())
}
