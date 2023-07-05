package network

const (
	ProtocolID        = "/doid/network/1"
	ProtocolGetBlocks = "/doid/block/get/1"
	ProtocolState     = "/doid/state/1"

	TopicPeer  = "/doid/topic/peer/1"
	TopicBlock = "/doid/topic/block/1"
)

var (
	ListenHost = "0.0.0.0"
	ListenPort = "3001"
)

const prefixCMDLength = 12
const versionInfo = byte(0x00)

type command string

const (
	cGetHash     command = "getHash"
	cHashMap     command = "hashMap"
	cTransaction command = "transaction"
	cMyError     command = "myError"
	cMyTest      command = "myTest"
)

const (
	metaVersion = "v"
	metaState   = "s"
)
