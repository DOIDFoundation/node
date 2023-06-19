package types

const (
	EventNewBlock = "NewBlock"
)

type ChainHeadEvent struct {
	// TODO
	Block *Block
}
