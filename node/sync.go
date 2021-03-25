package node

import (
	"context"
	"encoding/base64"
	"fmt"
	"ftp2p/state"
	"net/http"
	"time"

	"github.com/raphamorim/go-rainbow"
)

const endpointStatus = "/node/status"

const endpointSync = "/node/sync"
const endpointSyncQueryKeyFromBlock = "fromBlock"

const endpointAddPeer = "/node/peer"
const endpointAddPeerQueryKeyIP = "ip"
const endpointAddPeerQueryKeyPort = "port"
const endpointAddPeerQueryKeyMiner = "miner"
const endpointAddNameQueryKeyName = "name"

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(syncIntervalSeconds * time.Second)
	n.doSync()
	for {
		select {
		case <-ticker.C:
			n.doSync()

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", rainbow.Bold(rainbow.Green(peer.TcpAddress())))

		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", rainbow.Bold(rainbow.Green(peer.TcpAddress())))

			n.RemovePeer(peer)

			continue
		}

		err = n.joinKnownPeers(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncBlocks(peer, status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncKnownPeers(status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncPendingTXs(status.PendingTxs)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusResponse) error {
	localBlockNumber := n.state.LatestBlock().Header.Number
	// If the peer has no blocks, ignore it
	if status.Hash.IsEmpty() {
		return nil
	}

	// If the peer has less blocks than us, ignore it
	if status.Number < localBlockNumber {
		return nil
	}

	// If it's the genesis block and we already synced it, ignore it
	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	// Display found 1 new block if we sync the genesis block 0
	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	if newBlocksCount > 1 {
		fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount,
			rainbow.Bold(rainbow.Green(peer.TcpAddress())))
	}
	// blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlock().Header.Parent)
	// get blocks from the peer's latest block's parent
	blockHash := n.state.LatestBlock().Header.Parent
	// retrieve all blocks after the parent block
	blocks, err := fetchBlocksFromPeer(peer, blockHash)
	if err != nil {
		return err
	}
	for _, block := range blocks {
		s, _, err := n.state.AddBlock(block)
		if err != nil {
			if s != nil {
				n.state = s
			}
			return err
		}

		n.newSyncedBlocks <- block
	}

	return nil
}

func (n *Node) syncKnownPeers(status StatusResponse) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("Found new Peer %s\n", rainbow.Bold(rainbow.Green(statusPeer.TcpAddress())))

			n.AddPeer(statusPeer)
		}
	}

	return nil
}

func (n *Node) syncPendingTXs(txs []state.SignedTx) error {
	for _, tx := range txs {
		err := n.AddPendingTX(tx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	url := fmt.Sprintf(
		"http://%s%s?%s=%s&%s=%d&%s=%s&%s=%s&%s=%s",
		peer.TcpAddress(),
		endpointAddPeer,
		endpointAddPeerQueryKeyIP, n.info.IP,
		endpointAddPeerQueryKeyPort, n.info.Port,
		endpointAddPeerQueryKeyMiner, n.info.Address,
		endpointAddNameQueryKeyName, n.info.Name,
		"publicKey", base64.StdEncoding.EncodeToString([]byte(n.info.EncryptionPublicKey)),
	)

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	addPeerRes := AddPeerRes{}
	err = readRes(res, &addPeerRes)
	if err != nil {
		return err
	}
	if addPeerRes.Error != "" {
		return fmt.Errorf(addPeerRes.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = addPeerRes.Success

	n.AddPeer(knownPeer)

	if !addPeerRes.Success {
		return fmt.Errorf("unable to join KnownPeers of '%s'", rainbow.Bold(rainbow.Green(peer.TcpAddress())))
	}

	return nil
}

func queryPeerStatus(peer PeerNode) (StatusResponse, error) {
	url := fmt.Sprintf("http://%s%s", peer.TcpAddress(), endpointStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusResponse{}, err
	}

	StatusResponse := StatusResponse{}
	err = readRes(res, &StatusResponse)
	if err != nil {
		return StatusResponse, err
	}

	return StatusResponse, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock state.Hash) ([]state.Block, error) {
	fmt.Printf("Importing blocks from Peer %s...\n", rainbow.Bold(rainbow.Green(peer.TcpAddress())))

	url := fmt.Sprintf(
		"http://%s%s?%s=%s",
		peer.TcpAddress(),
		endpointSync,
		endpointSyncQueryKeyFromBlock,
		fromBlock.Hex(),
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	syncRes := SyncRes{}
	err = readRes(res, &syncRes)
	if err != nil {
		return nil, err
	}

	return syncRes.Blocks, nil
}
