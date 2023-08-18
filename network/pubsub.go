package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/DOIDFoundation/node/config"
	"github.com/cometbft/cometbft/libs/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// @todo Compatible with legacy testnet topics, remove these later
type topicsWrapper struct {
	topics []*pubsub.Topic
}

// @todo Compatible with legacy testnet topics, remove these later
func (w *topicsWrapper) Publish(ctx context.Context, data []byte, opts ...pubsub.PubOpt) error {
	for _, topic := range w.topics {
		if err := topic.Publish(ctx, data, opts...); err != nil {
			return err
		}
	}
	return nil
}

// @todo Compatible with legacy testnet topics, remove these later
func (w *topicsWrapper) ListPeers() (peers []peer.ID) {
	for _, topic := range w.topics {
		peers = append(peers, topic.ListPeers()...)
	}
	return
}

// @todo Compatible with legacy testnet topics, remove these later
type topicWrapper interface {
	Publish(ctx context.Context, data []byte, opts ...pubsub.PubOpt) error
	ListPeers() []peer.ID
}

// @todo Compatible with legacy testnet topics, remove these later
// Join both /doid/2/... and /doid/... topics to be compatible with topics without chainid
func joinTestnetTopics(topic string, topicHandler func(string) *pubsub.Topic) topicWrapper {
	t := topicHandler(topic)
	if t == nil {
		return nil
	}

	newTopicPrefix := fmt.Sprintf("/doid/%d/", config.NetworkID)
	const legacyTopicPrefix = "/doid/"
	if strings.HasPrefix(topic, newTopicPrefix) {
		w := new(topicsWrapper)
		w.topics = append(w.topics, t)
		legacyTopic := strings.Replace(topic, newTopicPrefix, legacyTopicPrefix, 1)
		if topic := topicHandler(legacyTopic); t != nil {
			w.topics = append(w.topics, topic)
		}
		return w
	}
	return t
}

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
