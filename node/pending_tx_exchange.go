package node

import (
	"context"
	"encoding/json"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/libp2p/go-libp2p-core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// PendingTxBufSize is the number of incoming pending transactions to buffer for each epoch.
const PendingTxBufSize = 128

// const PENDING_TX_TOPIC = "PENDING_TX_TOPIC"

// Channel represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Channel.Publish, and received
// messages are pushed to the Messages channel.
type PendingTransactionsExchange struct {
	// A channel of signed transactions to send new pending transactions to peers
	PendingTransactions chan *state.SignedTx
	ctx                 context.Context
	ps                  *pubsub.PubSub
	topic               *pubsub.Topic
	sub                 *pubsub.Subscription
	self                peer.ID
}

// JoinPendingTxExchange tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinPendingTxExchange(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID) (*PendingTransactionsExchange, error) {
	topic, err := ps.Join(PENDING_TX_TOPIC)
	if err != nil {
		return nil, err
	}
	// TODO: can I add an event handler here to handle new pending tx events?
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	cr := &PendingTransactionsExchange{
		ctx:                 ctx,
		ps:                  ps,
		topic:               topic,
		sub:                 sub,
		self:                selfID,
		PendingTransactions: make(chan *state.SignedTx, PendingTxBufSize),
	}
	// start reading messages from the subscription in a loop
	go cr.readLoop()
	return cr, nil
}

// Publish sends a message to the pubsub topic.
func (cr *PendingTransactionsExchange) Publish(tx *state.SignedTx) error {
	msgBytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *PendingTransactionsExchange) ListPeers() []peer.ID {
	return cr.ps.ListPeers(PENDING_TX_TOPIC)
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *PendingTransactionsExchange) readLoop() {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			close(cr.PendingTransactions)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}
		cm := new(state.SignedTx)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid messages onto the Messages channel
		cr.PendingTransactions <- cm
	}
}
