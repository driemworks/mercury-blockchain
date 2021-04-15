package node

import (
	"context"
	"encoding/json"
	"fmt"
	"ftp2p/core"
	"ftp2p/state"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

const miningIntervalSeconds = 45
const syncIntervalSeconds = 30

type Node struct {
	datadir         string
	ip              string
	port            uint64
	state           *state.State
	info            core.PeerNode
	knownPeers      map[string]core.PeerNode
	trustedPeers    map[string]core.PeerNode
	pendingTXs      map[string]state.SignedTx
	archivedTXs     map[string]state.SignedTx
	newSyncedBlocks chan state.Block
	newPendingTXs   chan state.SignedTx
	isMining        bool
	name            string
}

func NewNode(name string, datadir string, ip string, port uint64,
	address common.Address, bootstrap core.PeerNode) *Node {
	knownPeers := make(map[string]core.PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap
	return &Node{
		name:            name,
		datadir:         datadir,
		ip:              ip,
		port:            port,
		knownPeers:      knownPeers,
		trustedPeers:    make(map[string]core.PeerNode),
		info:            core.NewPeerNode(name, ip, port, false, address, true),
		pendingTXs:      make(map[string]state.SignedTx),
		archivedTXs:     make(map[string]state.SignedTx),
		newSyncedBlocks: make(chan state.Block),
		newPendingTXs:   make(chan state.SignedTx, 10000),
		isMining:        false,
	}
}

/**
* Start the node's HTTP client
 */
func (n *Node) Run(ctx context.Context) error {
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.info.IP, n.info.Port))
	state, err := state.NewStateFromDisk(n.datadir)
	// trusted peers will need to be tracked as transactions, so that we can recover/rebuild when starting node?
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.sync(ctx)
	go n.mine(ctx)
	// publish a new CID
	http.HandleFunc("/cid", func(w http.ResponseWriter, r *http.Request) {
		sendCIDHandler(w, r, n)
	})
	// add a PeerNode to the trusted peers slice
	http.HandleFunc("/trusted-peer", func(w http.ResponseWriter, r *http.Request) {
		addTrustedPeerNodeHandler(w, r, n)
	})
	/*
		ENCRYPTION OPERATIONS
	*/
	// for now, only allow string data, but change that in the future
	http.HandleFunc("/encrypt", func(w http.ResponseWriter, r *http.Request) {
		encryptDataHandler(w, r, n)
	})
	// decrypt some data
	http.HandleFunc("/decrypt", func(w http.ResponseWriter, r *http.Request) {
		decryptDataHandler(w, r, n)
	})
	/*
		READ OPERATIONS
	*/
	http.HandleFunc("/received", func(w http.ResponseWriter, r *http.Request) {
		inboxHandler(w, r, n)
	})
	http.HandleFunc("/sent", func(w http.ResponseWriter, r *http.Request) {
		sentHandler(w, r, n)
	})
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		infoHandler(w, r, n)
	})
	http.HandleFunc("/peers/known", func(w http.ResponseWriter, r *http.Request) {
		knownPeersHandler(w, r, n)
	})
	http.HandleFunc("/peers/trusted", func(w http.ResponseWriter, r *http.Request) {
		trustedPeersHandler(w, r, n)
	})
	// THE BELOW COULD BE RPC?
	// get node status
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, n)
	})
	// peer/block/tx sync
	http.HandleFunc("/node/sync", func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})
	// add to known peers
	http.HandleFunc("/node/peer", func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", n.port)}
	// s.Stop()
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
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

func (n *Node) AddPeer(peer core.PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer core.PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer core.PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
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

		fmt.Printf("Adding pending transactions: \n%s\n", &prettyTxJSON)

		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
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
