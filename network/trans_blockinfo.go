package network

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func (n *Network) registerBlockInfoSubscribers() {
	logger := n.Logger.With("topic", "block_info")
	topic, err := n.pubsub.Join("/doid/block_info")
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
	n.topicBlockInfo = topic
	logger.Info("topic joined")

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		data := msg.GetData()
		blockInfo := new(BlockInfo)
		err := rlp.DecodeBytes(data, blockInfo)
		if err != nil {
			logger.Error("failed to decode received block", "err", err)
			return
		}
		logger.Debug("got message block info", "height", blockInfo.Height, "peer", msg.GetFrom())

		store := n.localHost.Peerstore()
		store.Put(msg.GetFrom(), "height", blockInfo.Height)
		if n.networkHeight.Cmp(blockInfo.Height) < 0 {
			n.networkHeight.Set(blockInfo.Height)
		}

		switch n.blockChain.LatestBlock().Header.Height.Cmp(blockInfo.Height) {
		case -1:
			logger.Info("we are behind, start sync", "ours", n.blockChain.LatestBlock().Header.Height)
			n.publishBlockGet(big.NewInt(0).Add(n.blockChain.LatestBlock().Header.Height, common.Big1))
		case 0:
		case 1:
			peerNotifier <- msg.GetFrom().String()
		}
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

			if msg.ReceivedFrom == n.localHost.ID() {
				continue
			}
			logger.Debug("got msg", "peer", msg.GetFrom())
			go pipeline(msg)
		}
	}

	go messageLoop()
}
