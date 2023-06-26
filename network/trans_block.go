package network

import (
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"math/big"
)

func (n *Network) registerBlockSubscribers() {
	topic, err := n.pubsub.Join("/doid/blocks")
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
	n.topicBlock = topic
	n.Logger.Info("create topic blocks")

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		block := new(types.Block)
		err := rlp.DecodeBytes(data, block)
		if err != nil {
			n.Logger.Error("failed to decode received block", "err", err)
			return
		}
		n.Logger.Info("got message blocks", "block height", block.Header.Height)

		if n.blockChain.LatestBlock().Header.Height.Int64() < block.Header.Height.Int64() {
			n.Logger.Info("apply block", "block height", block.Header.Height)
			err = n.blockChain.ApplyBlock(block)
			if err == nil {
				if n.blockChain.LatestBlock().Header.Height.Uint64() < maxHeight {
					blockHeight := BlockHeight{Height: big.NewInt(n.blockChain.LatestBlock().Header.Height.Int64() + 1)}
					b, err := rlp.EncodeToBytes(blockHeight)
					if err != nil {
						n.Logger.Error("failed to encode block for broadcasting", "err", err)
						return
					}
					n.topicBlockGet.Publish(ctx, b)
				}
			}

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

			n.Logger.Info("topic blocks msg from ", msg.ReceivedFrom)
			if msg.ReceivedFrom == n.localHost.ID() {
				continue
			}

			go pipeline(msg)
		}
	}

	go messageLoop()
}
