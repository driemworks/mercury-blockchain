package node

import (
	"context"
	"driemcoin/main/manifest"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// const httpPort = 8080
const miningIntervalSeconds = 10

type PeerNode struct {
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
	pendingTXs      map[string]manifest.SignedTx
	archivedTXs     map[string]manifest.SignedTx
	newSyncedBlocks chan manifest.Block
	newPendingTXs   chan manifest.SignedTx
	isMining        bool
	alias           string
}

func NewNode(alias string, datadir string, ip string, port uint64, address common.Address, boostrap PeerNode) *Node {
	knownPeers := make(map[string]PeerNode)
	knownPeers[boostrap.TcpAddress()] = boostrap
	return &Node{
		alias:           alias,
		datadir:         datadir,
		ip:              ip,
		port:            port,
		knownPeers:      knownPeers,
		info:            NewPeerNode(ip, port, false, address, true),
		pendingTXs:      make(map[string]manifest.SignedTx),
		archivedTXs:     make(map[string]manifest.SignedTx),
		newSyncedBlocks: make(chan manifest.Block),
		newPendingTXs:   make(chan manifest.SignedTx, 10000),
		isMining:        false,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, address common.Address, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, address, connected}
}

/**
* Start the node's HTTP client
 */
func (n *Node) Run(ctx context.Context) error {
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.info.IP, n.info.Port))
	state, err := manifest.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.sync(ctx)
	go n.mine(ctx)

	// list manifest
	http.HandleFunc("/manifest/list", func(w http.ResponseWriter, r *http.Request) {
		listManifestHandler(w, r, state)
	})
	// send CID to someone
	http.HandleFunc("/cid/add", func(w http.ResponseWriter, r *http.Request) {
		addCIDHandler(w, r, n)
	})
	// get the nodes' status
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, n)
	})
	/* sync endpoints */
	// peer sync
	http.HandleFunc("/node/sync", func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})
	// block sync
	http.HandleFunc("/node/peer", func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", n.port)}
	// channels are weird..
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
						fmt.Printf("ERROR: %s\n", err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", blockHash.Hex())

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

	_, err = n.state.AddBlock(minedBlock)
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
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

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
func (n *Node) AddPendingTX(tx manifest.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		fmt.Printf("Added Pending TX %s from Peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
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
