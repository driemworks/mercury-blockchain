package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/driemworks/mercury-blockchain/core"
	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"

	"github.com/raphamorim/go-rainbow"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(syncIntervalSeconds * time.Second)
	n.doSync(ctx)
	for {
		select {
		case <-ticker.C:
			n.doSync(ctx)

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync(ctx context.Context) {
	for _, peer := range n.knownPeers {
		// if you're the peer then do nothing
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		// create a client connection to the peer's server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := RunRPCClient(ctx, peer.TcpAddress(), n.tls, "", "")
		if err != nil {
			handleErrRemovePeer(err, n, peer)
			continue
		}
		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n",
			rainbow.Bold(rainbow.Green(peer.TcpAddress())))
		// query the peer node's status to sync name
		info, err := queryNodeStatus(ctx, client, &peer)
		if err != nil {
			handleErrRemovePeer(err, n, peer)
			continue
		}
		// sync the name
		peer.Name = info.Name
		err = n.joinKnownPeers(ctx, client, peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		// sync blocks
		err = n.syncBlocks(ctx, client, peer, info)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		// sync known peers
		err = queryKnownPeers(ctx, client, n, peer)
		if err != nil {
			handleErrRemovePeer(err, n, peer)
			continue
		}
		// sync pending txs
		err = n.syncPendingTXs(ctx, client)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func handleErrRemovePeer(err error, n *Node, peer core.PeerNode) {
	fmt.Printf("ERROR: %s\n", err)
	fmt.Printf("Peer '%s' was removed from KnownPeers\n",
		rainbow.Bold(rainbow.Green(peer.TcpAddress())))
	n.RemovePeer(peer)
}

func (n *Node) syncPendingTXs(ctx context.Context, client pb.PublicNodeClient) error {
	stream, err := client.ListPendingTransactions(ctx, &pb.ListPendingTransactionsRequest{})
	if err != nil {
		log.Fatalf("%v.ListPendingTransactions(_) = _, %v", client, err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListPendingTransactions(_) = _, %v", client, err)
		}
		// TODO: Tx amount is hard coded as 0
		var signedTx state.SignedTx
		err = json.Unmarshal(res.SignedTx, &signedTx)
		if err != nil {
			log.Fatalf("%v.ListPendingTransactions(_) = _, %v", client, err)
		}
		n.AddPendingTX(signedTx)
	}
	return nil
}

func (n *Node) syncBlocks(ctx context.Context, client pb.PublicNodeClient, peer core.PeerNode, peerNodeInfo pb.NodeInfoResponse) error {
	localBlockNumber := n.state.LatestBlock().Header.Number
	// If the peer has no blocks, ignore it
	if peerNodeInfo.Hash == "" {
		return nil
	}

	// If the peer has less blocks than us, ignore it
	if peerNodeInfo.Number < localBlockNumber {
		return nil
	}

	// If it's the genesis block and we already synced it, ignore it
	if peerNodeInfo.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	// Display found 1 new block if we sync the genesis block 0
	newBlocksCount := peerNodeInfo.Number - localBlockNumber
	if localBlockNumber == 0 && peerNodeInfo.Number == 0 {
		newBlocksCount = 1
	}
	if newBlocksCount > 1 {
		fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount,
			rainbow.Bold(rainbow.Green(peer.TcpAddress())))
	}
	// blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlock().Header.Parent)
	// get blocks from the peer's latest block's parent
	blockHash := n.state.LatestBlockHash()
	// TODO - should really fetch from the latest block's parent. This is to account for the scenario where multiple nodes mine the same block
	// before they can sync

	stream, err := client.ListBlocks(ctx, &pb.ListBlocksRequest{
		FromBlock: blockHash.Hex(),
	})
	if err != nil {
		log.Fatalf("%v.ListBlocks(_) = _, %v", client, err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListBlocks(_) = _, %v", client, err)
		}
		var newBlock state.Block
		json.Unmarshal(res.Block, &newBlock)
		s, _, err := n.state.AddBlock(newBlock)
		if err != nil {
			if s != nil {
				n.state = s
			}
			return err
		}
		n.newSyncedBlocks <- newBlock
	}
	return nil
}

func queryNodeStatus(ctx context.Context, client pb.PublicNodeClient, pn *core.PeerNode) (pb.NodeInfoResponse, error) {
	res, err := client.GetNodeStatus(ctx, &pb.NodeInfoRequest{})
	if err != nil {
		log.Fatalf("%v.GetNodeStatus(_) = _, %v", client, err)
		return *&pb.NodeInfoResponse{}, err
	}
	return *res, nil
}

func queryKnownPeers(ctx context.Context, client pb.PublicNodeClient, n *Node, pn core.PeerNode) error {
	stream, err := client.ListKnownPeers(ctx, &pb.ListKnownPeersRequest{})
	if err != nil {
		log.Fatalf("%v.ListKnownPeers(_) = _, %v", client, err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListKnownPeers(_) = _, %v", client, err)
		}
		// if you don't know the peer, add it
		// TODO -> is it safe to assume that IsConnected can always be true?
		newPeer := core.NewPeerNode(res.Name, res.Ip, res.Port, res.IsBootstrap, state.NewAddress(res.Address), true)
		if !n.IsKnownPeer(newPeer) {
			fmt.Println("Adding new known peer!")
			n.knownPeers[newPeer.TcpAddress()] = newPeer
		}
	}
	return nil
}

func (n *Node) joinKnownPeers(ctx context.Context, client pb.PublicNodeClient, peer core.PeerNode) error {
	if peer.Connected {
		return nil
	}
	_, err := client.JoinKnownPeers(ctx, &pb.JoinKnownPeersRequest{
		Name:        n.name,
		Ip:          n.ip,
		Port:        n.port,
		Address:     n.info.Address.String(),
		IsBootstrap: n.info.IsBootstrap,
	})
	if err != nil {
		return err
	}
	return nil
}
