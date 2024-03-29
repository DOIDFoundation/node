package network

import (
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/rlp"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func (n *Network) topicTxHandler(TopicTx string) *pubsub.Topic {
	logger := n.Logger.With("topic", TopicTx)
	topic, err := n.pubsub.Join(TopicTx)
	if err != nil {
		logger.Error("Failed to join pubsub topic", "err", err)
		return nil
	}

	onTx := func(msg *pubsub.Message) {
		data := msg.GetData()
		tx := new(types.Tx)
		if err := rlp.DecodeBytes(data, tx); err != nil {
			logger.Error("failed to decode received tx", "err", err)
			return
		}
		logger.Debug("got tx", "hash", tx.Hash(), "peer", msg.GetFrom())

		events.NewNetworkTx.Send(*tx)
	}

	go pubsubMessageLoop(ctx, topic, n.host.ID(), onTx, logger)
	return topic
}

func (n *Network) joinTopicTx() {
	// @todo Compatible with legacy testnet topics, recover this with topicTxHandler later
	n.topicTx = joinTestnetTopics(TopicTx, n.topicTxHandler)
}
