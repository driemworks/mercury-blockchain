package node

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/driemworks/mercury-blockchain/core"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/ethereum/go-ethereum/common"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/raphamorim/go-rainbow"
)

const miningIntervalSeconds = 45
const syncIntervalSeconds = 2

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "mercury-service-tag"
const DiscoveryServiceTag_PendingTxs = "pending_txs"

type Node struct {
	datadir         string
	ip              string
	port            uint64
	miner           common.Address
	state           *state.State
	info            core.PeerNode
	trustedPeers    map[string]core.PeerNode
	pendingTXs      map[string]state.SignedTx
	archivedTXs     map[string]state.SignedTx
	newSyncedBlocks chan state.Block
	newPendingTXs   chan state.SignedTx
	isMining        bool
	name            string
	tls             bool
	host            host.Host
}

func NewNode(name string, datadir string, miner string, ip string, port uint64, tls bool) *Node {
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
		newPendingTXs:   make(chan state.SignedTx, 10000),
		isMining:        false,
		tls:             tls,
	}
}

func (n *Node) Run(ctx context.Context, port int, peer string, name string) error {
	// load the state
	state, err := state.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.runRPCServer("", "")
	go n.mine(ctx)
	err = n.runLibp2pNode(ctx, port, peer, name)
	if err != nil {
		return err
	}
	return nil
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
						fmt.Printf(rainbow.Red("ERROR: %s\n"), err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", rainbow.Yellow(blockHash.Hex()))

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Address,
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

	return nil
}

func (n *Node) removeMinedPendingTXs(block state.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\tArchiving mined TX: %s\n", rainbow.Yellow(txHash.Hex()))
			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

// func (n *Node) AddPeer(peer core.PeerNode) {
// 	n.knownPeers[peer.TcpAddress()] = peer
// }

// func (n *Node) RemovePeer(peer core.PeerNode) {
// 	delete(n.knownPeers, peer.TcpAddress())
// }

// func (n *Node) IsKnownPeer(peer core.PeerNode) bool {
// 	if peer.IP == n.info.IP && peer.Port == n.info.Port {
// 		return true
// 	}

// 	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

// 	return isKnownPeer
// }

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

		fmt.Printf("Adding pending transactions: \n%s\n", &prettyTxJSON)

		n.pendingTXs[txHash.Hex()] = tx
		// n.newPendingTXs <- tx
		tmpFrom := n.state.Manifest[tx.From]
		if tmpFrom.Sent == nil {
			tmpFrom.Inbox = make([]state.InboxItem, 0)
			tmpFrom.Sent = make([]state.SentItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpFrom.Balance += manifest.BlockReward
			// tmpFrom.PendingBalance += tmpFrom.Balance
		}
		// TODO - the cost of the transaction is one coin for now, but should this always be the case?
		//         could file size factor into the cost? -> maybe when I get to the concept of gas?
		if tx.Amount > 0 {
			tmpFrom.PendingBalance -= tx.Amount
		}
		n.state.Manifest[tx.From] = tmpFrom
		// increase the account to nonce value => allows us to support mining blocks with multiple transactions
		n.state.PendingAccount2Nonce[tx.From]++
		tmpTo := n.state.Manifest[tx.To]
		// needed?
		if tmpTo.Inbox == nil {
			tmpTo.Sent = make([]state.SentItem, 0)
			tmpTo.Inbox = make([]state.InboxItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpTo.Balance += manifest.BlockReward
			// tmpTo.PendingBalance += tmpTo.Balance
		}
		tmpTo.PendingBalance += tx.Amount
		n.state.Manifest[tx.To] = tmpTo
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
