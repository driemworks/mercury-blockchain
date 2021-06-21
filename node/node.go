package node

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/driemworks/mercury-blockchain/core"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/raphamorim/go-rainbow"
)

const miningIntervalSeconds = 15

// TODO this is getting messy
type Node struct {
	datadir         string
	ip              string
	port            uint64
	miner           common.Address
	state           *state.State
	pendingTXs      map[string]state.SignedTx
	archivedTXs     map[string]state.SignedTx
	newSyncedBlocks chan state.Block
	candidateBlocks chan state.Block
	newMinedBlocks  chan core.MessageTransport
	newPendingTXs   chan core.MessageTransport
	isMining        bool
	name            string
	tls             bool
	host            host.Host
	pubsub          *pubsub.PubSub
	stake           int
}

func NewNode(name string, datadir string, miner string, ip string, port uint64, tls bool, stake int) *Node {
	minerAddress := state.NewAddress(miner)
	return &Node{
		name:            name,
		datadir:         datadir,
		miner:           minerAddress,
		ip:              ip,
		port:            port,
		pendingTXs:      make(map[string]state.SignedTx),
		archivedTXs:     make(map[string]state.SignedTx),
		newSyncedBlocks: make(chan state.Block),
		newMinedBlocks:  make(chan core.MessageTransport),
		newPendingTXs:   make(chan core.MessageTransport, 10000),
		isMining:        false,
		tls:             tls,
		stake:           stake, // todo this is temporary, will remove when endpoint + validations added
	}
}

func (n *Node) Run(ctx context.Context, ip string, port int, rpcHost string, rpcPort uint64, peer string, name string) error {
	state, err := state.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go func() {
		err := n.runRPCServer("", "", rpcHost, rpcPort)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	go func() {
		err := n.mine(ctx)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	err = n.runLibp2pNode(ctx, ip, port, peer, name)
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) consensus(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * miningIntervalSeconds)
	for {
		select {
		case <-ticker.C:
			go func() {
				if n.stake > 0 && len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true // still needed?
					blockCreator, err := n.electBlockCreator()
					if err != nil {
						logrus.Errorln(err)
					}
					logrus.Infoln("Selected block creator ", blockCreator)
					// gossip selection to peers
					err = n.notifyBlockCreator(blockCreator)
					if err != nil {
						logrus.Errorln(err)
					}
				}
			}()
		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true
					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						logrus.Errorln(err)
					}
					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				logrus.Infoln("Peer mined next Block '%s' faster :(\n", rainbow.Yellow(blockHash.Hex()))
				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}
		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) notifyBlockCreator(b *common.Address) error {
	return nil
}

func (n *Node) electBlockCreator() (*common.Address, error) {
	return nil, nil
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := state.NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.miner,
		n.getPendingTXsAsArray(),
	)
	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)
	_, _, err = n.state.AddBlock(minedBlock)
	if err != nil {
		return err
	}
	blockBytes, err := json.Marshal(minedBlock)
	if err != nil {
		return err
	}
	n.newMinedBlocks <- core.MessageTransport{blockBytes}
	return nil
}

func (n *Node) removeMinedPendingTXs(block state.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		logrus.Infoln("Updating in-memory Pending TXs Pool")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			logrus.Infof("Archiving mined TX: %s\n", rainbow.Yellow(txHash.Hex()))
			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

/**
Add a pending transaction to the node's pending transactions array
*/
func (n *Node) AddPendingTX(tx state.SignedTx) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		txJSON, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		prettyTxJSON, err := core.PrettyPrintJSON(txJSON)
		if err != nil {
			return err
		}

		logrus.Infof("Adding pending transactions: \n%s\n", &prettyTxJSON)
		tmpFrom := n.state.Catalog[tx.Author]
		// if tmpFrom.Balance <= 0 {
		// 	// for now...
		// 	tmpFrom.Balance = 10
		// 	// return fmt.Errorf("Insufficient balance")
		// }
		tmpFrom.Balance -= 1
		n.pendingTXs[txHash.Hex()] = tx
		n.state.Catalog[tx.Author] = tmpFrom
		n.state.PendingAccount2Nonce[tx.Author]++
	}
	return nil
}

func (n *Node) getPendingTXsAsArray() []state.SignedTx {
	txs := make([]state.SignedTx, len(n.pendingTXs))
	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}
	return txs
}

func (n *Node) Join(ctx context.Context, topicName string,
	bufSize int, onMessage core.MessageHandler,
	msgChan chan core.MessageTransport) error {
	topic, err := n.pubsub.Join(topicName)
	if err != nil {
		return err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}
	ch := &core.PubSubWrapper{
		Topic: topic,
		Sub:   sub,
		Self:  n.host.ID(),
		Data:  make(chan core.MessageTransport, bufSize),
	}
	go ch.ReadLoop(ctx, onMessage)
	go ch.Publish(ctx, msgChan)
	return nil
}
