package node

import (
	"context"
	"encoding/json"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const PENDING_TX_TOPIC = "PENDING_TX_TOPIC"
const NEW_BLOCKS_TOPIC = "NEW_BLOCKS_TOPIC"

// Channel represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Channel.Publish, and received
// messages are pushed to the Messages channel.
type Channel struct {
	// A channel of signed transactions to send new pending transactions to peers
	data      chan map[string]interface{}
	topicName string
	ctx       context.Context
	ps        *pubsub.PubSub
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
	self      peer.ID
}

func InitChannel(ctx context.Context, topicName string, bufSize int, ps *pubsub.PubSub, selfID peer.ID) (*Channel, error) {
	topic, err := ps.Join(topicName)
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	ch := &Channel{
		ctx:   ctx,
		ps:    ps,
		topic: topic,
		sub:   sub,
		self:  selfID,
		data:  make(chan map[string]interface{}, bufSize),
	}
	// start reading messages from the subscription in a loop
	go ch.readLoop()
	return ch, nil
}

// Publish sends a message to the pubsub topic.
func (cr *Channel) Publish(tx *state.SignedTx) error {
	msgBytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *Channel) ListPeers() []peer.ID {
	return cr.ps.ListPeers(cr.topicName)
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *Channel) readLoop() {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			close(cr.data)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}
		cm := new(map[string]interface{})
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid txs to data chan
		cr.data <- *cm
	}
}
