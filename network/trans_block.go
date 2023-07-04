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
	logger := n.Logger.With("topic", "blocks")
	topic, err := n.pubsub.Join("/doid/blocks")
	if err != nil {
		logger.Error("Failed to join pubsub topic", "err", err)
		return
	}
	sub, err := topic.Subscribe()
	if err != nil {
		topic.Close()
		logger.Error("Failed to subscribe to pubsub topic", "err", err)
		return
	}
	n.topicBlock = topic
	logger.Info("create topic blocks")

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
		logger.Debug("got message blocks", "height", blockEvent.Block.Header.Height, "td", blockEvent.Td, "peer", msg.GetFrom())
		n.host.Peerstore().Put(msg.GetFrom(), metaVersion,
			peerState{Height: blockEvent.Block.Header.Height.Uint64(),
				Td: blockEvent.Td,
				ID: msg.GetFrom().String()})

		events.NewNetworkBlock.Send(blockEvent.Block)
	}

	// The main message loop for receiving incoming messages from this subscription.
	messageLoop := func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				// This should only happen when the context is cancelled or subscription is cancelled.
				if err != pubsub.ErrSubscriptionCancelled { // Only log an error on unexpected errors.
					logger.Error("Subscription next failed", "err", err)
				}
				// Cancel subscription in the event of an error, as we are
				// now exiting topic event loop.
				sub.Cancel()
				return
			}

			if msg.ReceivedFrom == n.host.ID() {
				continue
			}

			go pipeline(msg)
		}
	}

	go messageLoop()
}
