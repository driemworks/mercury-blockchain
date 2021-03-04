package node

import (
	"context"
	"encoding/json"
	"fmt"
	"ftp2p/logging"
	"ftp2p/manifest"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

const miningIntervalSeconds = 10
const syncIntervalSeconds = 30

type PeerNode struct {
	Name        string         `json:"name"`
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Address     common.Address `json:"address"`
	connected   bool
}

func (p PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

type Node struct {
	datadir         string
	ip              string
	port            uint64
	state           *manifest.State
	info            PeerNode
	knownPeers      map[string]PeerNode
	trustedPeers    map[string]PeerNode
	pendingTXs      map[string]manifest.SignedTx
	archivedTXs     map[string]manifest.SignedTx
	newSyncedBlocks chan manifest.Block
	newPendingTXs   chan manifest.SignedTx
	isMining        bool
	name            string
	// wallet          wallet.Wallet
}

func NewNode(name string, datadir string, ip string, port uint64, address common.Address, bootstrap PeerNode) *Node {
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap
	return &Node{
		name:            name,
		datadir:         datadir,
		ip:              ip,
		port:            port,
		knownPeers:      knownPeers,
		trustedPeers:    make(map[string]PeerNode),
		info:            NewPeerNode(name, ip, port, false, address, true),
		pendingTXs:      make(map[string]manifest.SignedTx),
		archivedTXs:     make(map[string]manifest.SignedTx),
		newSyncedBlocks: make(chan manifest.Block),
		newPendingTXs:   make(chan manifest.SignedTx, 10000),
		isMining:        false,
	}
}

func NewPeerNode(name string, ip string, port uint64, isBootstrap bool, address common.Address, connected bool) PeerNode {
	return PeerNode{name, ip, port, isBootstrap, address, connected}
}

/**
* Start the node's HTTP client
 */
func (n *Node) Run(ctx context.Context) error {
	// s := spinner.New(spinner.CharSets[9], 10*time.Millisecond)
	// s.Start()
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.info.IP, n.info.Port))
	state, err := manifest.NewStateFromDisk(n.datadir)
	// fmt.Println("Succesfully loaded the state from disk")

	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.sync(ctx)
	go n.mine(ctx)

	http.HandleFunc("/inbox", func(w http.ResponseWriter, r *http.Request) {
		inboxHandler(w, r, n)
	})
	http.HandleFunc("/sent", func(w http.ResponseWriter, r *http.Request) {
		sentHandler(w, r, n)
	})
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		infoHandler(w, r, n)
	})
	// send tokens to an address
	http.HandleFunc("/send/tokens", func(w http.ResponseWriter, r *http.Request) {
		sendTokensHandler(w, r, n)
	})
	// send CID to someone (costs 1 FTC)
	http.HandleFunc("/send/cid", func(w http.ResponseWriter, r *http.Request) {
		sendCIDHandler(w, r, n)
	})
	// add a PeerNode to the trusted peers slice
	http.HandleFunc("/peers/known", func(w http.ResponseWriter, r *http.Request) {
		knownPeersHandler(w, r, n)
	})
	// add a PeerNode to the trusted peers slice
	http.HandleFunc("/peers/trusted", func(w http.ResponseWriter, r *http.Request) {
		trustedPeersHandler(w, r, n)
	})
	// add a PeerNode to the trusted peers slice
	http.HandleFunc("/peers/trusted/add", func(w http.ResponseWriter, r *http.Request) {
		addTrustedPeerNodeHandler(w, r, n)
	})
	// for now, only allow string data, but change that in the future
	http.HandleFunc("/encrypt", func(w http.ResponseWriter, r *http.Request) {
		encryptDataHandler(w, r, n)
	})
	// decrypt some data
	http.HandleFunc("/decrypt", func(w http.ResponseWriter, r *http.Request) {
		decryptDataHandler(w, r, n)
	})

	// THE BELOW COULD BE TRANSLATED TO BE RPC
	// get the nodes' status
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, n)
	})
	/* sync endpoints => these should not be able to be called without some proper auth */
	// peer sync
	http.HandleFunc("/node/sync", func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})
	// block sync
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

func (n *Node) removeMinedPendingTXs(block manifest.Block) {
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

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}

/**
*
 */
func (n *Node) AddPendingTX(tx manifest.SignedTx) error {
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
		prettyTxJSON, err := logging.PrettyPrintJSON(txJSON)
		if err != nil {
			return err
		}

		fmt.Printf("Adding pending transactions: \n%s\n", &prettyTxJSON)

		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
		tmpFrom := n.state.Manifest[tx.From]
		// if this is the first pending transaction
		// TODO - checking if Sent is nil isn't a great thing.. should add a func to check if empty
		// this is what "requestToken" should do....
		if tmpFrom.Sent == nil {
			tmpFrom.Inbox = make([]manifest.InboxItem, 0)
			tmpFrom.Sent = make([]manifest.SentItem, 0)
			// tmpFrom.Balance += manifest.BlockReward
			// tmpFrom.PendingBalance += tmpFrom.Balance
		}
		// TODO - the cost of the transaction is one coin for now, but should this always be the case?
		//         could file size factor into the cost? -> maybe when I get to the concept of gas?
		if tx.Amount > 0 {
			tmpFrom.PendingBalance -= tx.Amount
		}
		n.state.Manifest[tx.From] = tmpFrom
		// increase the account nonce => allows us to support mining blocks with multiple transactions
		n.state.PendingAccount2Nonce[tx.From]++
		tmpTo := n.state.Manifest[tx.To]
		// if this is the first pending transaction, initialize data -> should this really happen this way? seems wrong...
		if tmpTo.Inbox == nil {
			tmpTo.Sent = make([]manifest.SentItem, 0)
			tmpTo.Inbox = make([]manifest.InboxItem, 0)
			// tmpTo.Balance += manifest.BlockReward
			// tmpTo.PendingBalance += tmpTo.Balance
		}
		tmpTo.PendingBalance += tx.Amount
		n.state.Manifest[tx.To] = tmpTo
	}
	return nil
}

func (n *Node) getPendingTXsAsArray() []manifest.SignedTx {
	txs := make([]manifest.SignedTx, len(n.pendingTXs))
	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}
	return txs
}
