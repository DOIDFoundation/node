package network

var (
	RendezvousString = "meetme"
	ProtocolID       = "/doid/network/0.0.1"
	ListenHost       = "0.0.0.0"
	ListenPort       = "3001"
)

const prefixCMDLength = 12
const versionInfo = byte(0x00)

type command string

const (
	cVersion     command = "version"
	cGetHash     command = "getHash"
	cHashMap     command = "hashMap"
	cGetBlock    command = "getBlock"
	cBlock       command = "block"
	cTransaction command = "transaction"
	cMyError     command = "myError"
	cMyTest      command = "myTest"
)
