package network

import (
	"context"

	"github.com/cometbft/cometbft/libs/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// The main message loop for receiving incoming messages from pubsub subscription.
func pubsubMessageLoop(ctx context.Context, topic *pubsub.Topic, self peer.ID, msgHandler func(*pubsub.Message), logger log.Logger) {
	sub, err := topic.Subscribe()
	if err != nil {
		topic.Close()
		logger.Error("Failed to subscribe to pubsub topic", "err", err)
		return
	}
	logger.Debug("pubsub topic subscribed")
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
			topic.Close()
			return
		}

		if msg.ReceivedFrom == self {
			continue
		}

		go msgHandler(msg)
	}
}
