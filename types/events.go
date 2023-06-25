package types

const (
	EventForkDetected    = "ForkDetected"
	EventNewChainHead    = "NewChainHead"
	EventNewMinedBlock   = "NewMinedBlock"
	EventNewNetworkBlock = "NewNetworkBlock"
	EventNewTx           = "NewTx"
	EventSyncStarted     = "SyncStarted"
	EventSyncFinished    = "SyncFinished"
)

type ChainHeadEvent struct {
	// TODO
	Block *Block
}
