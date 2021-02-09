package node

import (
	"context"
	"driemcoin/main/manifest"
	"fmt"
	"net/http"
	"time"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)

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
		if n.ip == peer.IP && n.port == peer.Port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", peer.TcpAddress())

		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", peer.TcpAddress())

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

		err = n.syncKnownPeers(peer, status)
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
	fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TcpAddress())

	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	return n.state.AddBlocks(blocks)
}

func (n *Node) syncKnownPeers(peer PeerNode, status StatusResponse) error {
	for _, statusPeer := range status.knownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("Found new Peer %s\n", statusPeer.TcpAddress())

			n.AddPeer(statusPeer)
		}
	}

	return nil
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	url := fmt.Sprintf(
		"http://%s%s?%s=%s&%s=%d",
		peer.TcpAddress(),
		"/node/peer",
		"ip",
		n.ip,
		"port",
		n.port,
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
		return fmt.Errorf("unable to join KnownPeers of '%s'", peer.TcpAddress())
	}

	return nil
}

func queryPeerStatus(peer PeerNode) (StatusResponse, error) {
	url := fmt.Sprintf("http://%s%s", peer.TcpAddress(), "/node/status")
	res, err := http.Get(url)
	if err != nil {
		return StatusResponse{}, err
	}

	statusRes := StatusResponse{}
	err = readRes(res, &statusRes)
	if err != nil {
		return StatusResponse{}, err
	}

	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock manifest.Hash) ([]manifest.Block, error) {
	fmt.Printf("Importing blocks from Peer %s...\n", peer.TcpAddress())

	url := fmt.Sprintf(
		"http://%s%s?%s=%s",
		peer.TcpAddress(),
		"/node/sync",
		"fromBlock",
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
