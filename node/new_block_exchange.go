package node

import (
	"context"
	"encoding/json"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/libp2p/go-libp2p-core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// TODO consider rethinking this design.. should use an interface?
// topic buf size is the number of incoming pending transactions to buffer for each epoch.
const TopicBufSize = 128

// const NEW_BLOCKS_TOPIC = "NEW_BLOCKS_TOPIC"

// Channel represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Channel.Publish, and received
// messages are pushed to the Messages channel.
type NewBlockExchange struct {
	// A channel of signed transactions to send new pending transactions to peers
	NewBlocks chan *state.Block
	ctx       context.Context
	ps        *pubsub.PubSub
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
	self      peer.ID
}

// Join tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinNewBlockExchange(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID) (*NewBlockExchange, error) {
	topic, err := ps.Join(NEW_BLOCKS_TOPIC)
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	cr := &NewBlockExchange{
		ctx:       ctx,
		ps:        ps,
		topic:     topic,
		sub:       sub,
		self:      selfID,
		NewBlocks: make(chan *state.Block, TopicBufSize),
	}
	// start reading messages from the subscription in a loop
	go cr.readLoop()
	return cr, nil
}

// Publish sends a message to the pubsub topic.
func (cr *NewBlockExchange) Publish(tx *state.Block) error {
	msgBytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *NewBlockExchange) ListPeers() []peer.ID {
	return cr.ps.ListPeers(PENDING_TX_TOPIC)
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *NewBlockExchange) readLoop() {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			close(cr.NewBlocks)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}
		cm := new(state.Block)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid messages onto the Messages channel
		cr.NewBlocks <- cm
	}
}
