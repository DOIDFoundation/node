package network

import (
	"bytes"
	"encoding/gob"
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func (n *Network) registerBlockHeightSubscribers() {
	// @todo use different topic for different fork
	topic, err := n.pubsub.Join("/doid/block_height")
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
	n.topicBlockHeight = topic

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		blockHeight := new(BlockHeightRequest)
		err := rlp.DecodeBytes(data, blockHeight)
		if err != nil {
			n.Logger.Error("failed to decode received block", "err", err)
			return
		}
		n.Logger.Debug("got message", "block height", blockHeight.BlockHeight)

		if n.chain.LatestBlock().Header.Height.Int64() < blockHeight.BlockHeight.Int64() {
			blockHeightRequest := BlockHeightRequest{BlockHeight: n.chain.LatestBlock().Header.Height + 1}
			b, err := rlp.EncodeToBytes(blockHeightRequest)
			if err != nil {
				n.Logger.Error("failed to encode block for broadcasting", "err", err)
				return
			}
			n.topicBlockHeight.Publish(ctx, b)
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
