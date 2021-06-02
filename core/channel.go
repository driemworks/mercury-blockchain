package core

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const PENDING_TX_TOPIC = "PENDING_TX_TOPIC"
const NEW_BLOCKS_TOPIC = "NEW_BLOCKS_TOPIC"

type MessageTransport struct {
	Data []byte
}

// PubSubWrapper represents a subscription to a single PubSub topic. Messages
// can be published to the topic with PubSubWrapper.Publish, and received
// messages are pushed to the Messages channel.
// TODO this name is bad... but works for now..
type PubSubWrapper struct {
	// A channel of signed transactions to send new pending transactions to peers
	Data      chan MessageTransport
	TopicName string
	Context   context.Context
	PubSub    *pubsub.PubSub
	Topic     *pubsub.Topic
	Sub       *pubsub.Subscription
	Self      peer.ID
}

// Publish sends a message to the pubsub topic.
func (cr *PubSubWrapper) Publish(ctx context.Context, msgChan chan MessageTransport) error {
	// publish loop
	for {
		select {
		case m := <-msgChan:
			err := cr.Topic.Publish(ctx, m.Data)
			if err != nil {
				return err
			}
		}
	}
}

func (cr *PubSubWrapper) ListPeers(ps *pubsub.PubSub) []peer.ID {
	return ps.ListPeers(cr.TopicName)
}

type MessageHandler func(data *pubsub.Message)
type PublishHandler func(msg []byte)

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *PubSubWrapper) ReadLoop(ctx context.Context, onMessage MessageHandler) {
	for {
		msg, err := cr.Sub.Next(ctx)
		if err != nil {
			fmt.Println(err)
			close(cr.Data)
			return
		}
		// only forward messages delivered by others
		// if msg.ReceivedFrom == cr.Self {
		// 	continue
		// }
		onMessage(msg)
	}
}
