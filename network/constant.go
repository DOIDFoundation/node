package network

const (
	ProtocolID        = "/doid/network/1"
	ProtocolGetBlocks = "/doid/block/get/1"
	ProtocolState     = "/doid/state/1"

	TopicPeer  = "/doid/topic/peer/1"
	TopicBlock = "/doid/topic/block/1"
	TopicTx    = "/doid/topic/tx/1"
)

const prefixCMDLength = 12
const versionInfo = byte(0x00)

type command string

const (
	cMyError command = "myError"
)

const (
	metaState = "s"
)
