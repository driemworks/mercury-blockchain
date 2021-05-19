package node

import (
	"context"

	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/driemworks/mercury-blockchain/wallet"
)

type nodeServer struct {
	pb.UnimplementedNodeServiceServer
	node *Node
}

func (server nodeServer) GetNodeStatus(ctx context.Context, statusRequest *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	nodeState := server.node.state.Catalog[server.node.miner]
	var channels []string
	for _, bytes := range nodeState.Channels {
		// this probably isn't right...
		channels = append(channels, string(bytes))
	}
	return &pb.NodeInfoResponse{
		Address:  server.node.miner.Hex(),
		Balance:  nodeState.Balance,
		Channels: channels,
	}, nil
}

/*
	Read/Write known peers
*/
// func (server publicNodeServer) ListKnownPeers(listKnownPeersRequest *pb.ListKnownPeersRequest,
// 	stream pb.PublicNode_ListKnownPeersServer) error {
// 	for tcp, pn := range server.node.knownPeers {
// 		if err := stream.Send(&pb.ListKnownPeersResponse{
// 			Name:        pn.Name,
// 			Ip:          server.node.knownPeers[tcp].IP,
// 			Port:        server.node.knownPeers[tcp].Port,
// 			IsBootstrap: pn.IsBootstrap,
// 			Address:     pn.Address.String(),
// 		}); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

/*
	Read
*/
// func (server publicNodeServer) ListBlocks(listBlocksRequest *pb.ListBlocksRequest,
// 	stream pb.PublicNode_ListBlocksServer) error {
// 	hash := state.Hash{}
// 	err := hash.UnmarshalText([]byte(listBlocksRequest.FromBlock))
// 	if err != nil {
// 		return err
// 	}
// 	blocks, err := state.GetBlocksAfter(hash, server.node.datadir)
// 	if err != nil {
// 		return err
// 	}

// 	for _, block := range blocks {
// 		encoded, err := json.Marshal(block)
// 		if err != nil {
// 			break
// 		}
// 		stream.Send(&pb.BlockResponse{
// 			Block: encoded,
// 		})
// 	}
// 	return nil
// }

/*
	Read/Write pending transactions
*/
func (server nodeServer) AddTransaction(
	ctx context.Context, addPendingTransactionRequest *pb.AddPendingTransactionRequest) (
	*pb.AddPendingTransactionResponse, error) {
	nonce := server.node.state.PendingAccount2Nonce[server.node.miner] + 1
	tx := state.NewTx(
		server.node.miner, addPendingTransactionRequest.Topic, nonce,
	)
	signedTx, err := wallet.SignTxWithKeystoreAccount(
		tx, server.node.miner, "test", wallet.GetKeystoreDirPath(server.node.datadir))
	if err != nil {
		return nil, err
	}
	server.node.AddPendingTX(signedTx)
	server.node.newPendingTXs <- signedTx
	return &pb.AddPendingTransactionResponse{}, nil
}

// // TODO for now this is only the publish cid tx... will generalize later
// func (server publicNodeServer) ListPendingTransactions(request *pb.ListPendingTransactionsRequest,
// 	stream pb.PublicNode_ListPendingTransactionsServer) error {
// 	for _, tx := range server.node.pendingTXs {
// 		bytes, err := json.Marshal(tx)
// 		if err != nil {
// 			return err
// 		}
// 		err = stream.Send(&pb.PendingTransactionResponse{
// 			SignedTx: bytes,
// 		})
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

func newNodeServer(n *Node) nodeServer {
	nodeServer := nodeServer{node: n}
	return nodeServer
}
