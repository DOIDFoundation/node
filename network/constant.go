package network

import (
	"fmt"

	"github.com/DOIDFoundation/node/config"
)

var ProtocolID,
	ProtocolGetBlocks,
	ProtocolState,

	TopicPeer,
	TopicBlock,
	TopicTx string

const (
	metaState = "s"
)

func initConstants() {
	networkId := config.NetworkID
	version := 1

	ProtocolID = fmt.Sprintf("/doid/%d/network/%d", networkId, version)
	ProtocolGetBlocks = fmt.Sprintf("/doid/%d/block/get/%d", networkId, version)
	ProtocolState = fmt.Sprintf("/doid/%d/state/%d", networkId, version)

	TopicPeer = fmt.Sprintf("/doid/%d/topic/peer/%d", networkId, version)
	TopicBlock = fmt.Sprintf("/doid/%d/topic/block/%d", networkId, version)
	TopicTx = fmt.Sprintf("/doid/%d/topic/tx/%d", networkId, version)
}
