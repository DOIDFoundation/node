package network

import (
	"github.com/DOIDFoundation/node/events"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	eventPeerState = &events.FeedOf[peer.ID]{}
)
