package types

const (
	EventNewChainHead    = "NewChainHead"
	EventNewMinedBlock   = "NewMinedBlock"
	EventNewNetworkBlock = "NewNetworkBlock"
)

type ChainHeadEvent struct {
	// TODO
	Block *Block
}
