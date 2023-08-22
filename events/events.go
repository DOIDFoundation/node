package events

import (
	"math/big"

	"github.com/DOIDFoundation/node/types"
)

var (
	ForkDetected    = &Feed{}                 // Found a block that indicates fork.
	NewChainHead    = &FeedOf[*types.Block]{} // Chain switched to a new head block.
	NewMinedBlock   = &FeedOf[*types.Block]{} // A new block is mined into current chain
	NewNetworkBlock = &FeedOf[BlockWithTd]{}  // A new block recieved from network
	NewNetworkTx    = &FeedOf[types.Tx]{}
	NewTx           = &FeedOf[types.Tx]{}
	SyncStarted     = &Feed{}
	SyncFinished    = &Feed{}
)

type BlockWithTd struct {
	*types.Block
	Td *big.Int
}
