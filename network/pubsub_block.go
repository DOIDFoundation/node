package network

import (
	"math/big"

	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type newBlock struct {
	Block *types.Block
	Td    *big.Int
}

func (n *Network) registerBlockSubscribers() {
	logger := n.Logger.With("topic", TopicBlock)
	topic, err := n.pubsub.Join(TopicBlock)
	if err != nil {
		logger.Error("Failed to join pubsub topic", "err", err)
		return
	}
	n.topicBlock = topic

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		blockEvent := new(newBlock)
		err := rlp.DecodeBytes(data, blockEvent)
		if err != nil {
			logger.Error("failed to decode received block", "err", err)
			return
		}
		logger.Debug("got block", "height", blockEvent.Block.Header.Height, "td", blockEvent.Td, "peer", msg.GetFrom())
		updatePeerState(n.host.Peerstore(), msg.GetFrom(), &state{Height: blockEvent.Block.Header.Height.Uint64(), Td: blockEvent.Td})

		events.NewNetworkBlock.Send(blockEvent.Block)
	}

	go pubsubMessageLoop(ctx, topic, n.host.ID(), pipeline, logger)
}
