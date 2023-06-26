package network

import (
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func (n *Network) registerBlockGetSubscribers() {
	// @todo use different topic for different fork
	topic, err := n.pubsub.Join("/doid/block_get")
	if err != nil {
		n.Logger.Error("Failed to join pubsub topic", "err", err)
		return
	}
	sub, err := topic.Subscribe()
	if err != nil {
		topic.Close()
		n.Logger.Error("Failed to subscribe to pubsub topic", "err", err)
		return
	}
	n.topicBlockGet = topic

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		blockHeight := new(BlockHeight)
		err := rlp.DecodeBytes(data, blockHeight)
		if err != nil {
			n.Logger.Error("failed to decode received block", "err", err)
			return
		}
		n.Logger.Debug("got message", "block height", blockHeight.Height)

		if n.chain.LatestBlock().Header.Height.Int64() > blockHeight.Height.Int64() {
			block := n.chain.BlockByHeight(blockHeight.Height.Uint64())

			b, err := rlp.EncodeToBytes(block)
			if err != nil {
				n.Logger.Error("failed to encode block for broadcasting", "err", err)
				return
			}
			n.topicBlock.Publish(ctx, b)
		}
	}

	// The main message loop for receiving incoming messages from this subscription.
	messageLoop := func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				// This should only happen when the context is cancelled or subscription is cancelled.
				if err != pubsub.ErrSubscriptionCancelled { // Only log an error on unexpected errors.
					n.Logger.Error("Subscription next failed", "err", err)
				}
				// Cancel subscription in the event of an error, as we are
				// now exiting topic event loop.
				sub.Cancel()
				return
			}

			if msg.ReceivedFrom == n.localHost.ID() {
				continue
			}

			go pipeline(msg)
		}
	}

	go messageLoop()
}
