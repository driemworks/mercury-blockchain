package node

import (
	"context"

	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/driemworks/mercury-blockchain/wallet"
)

type publicNodeServer struct {
	pb.UnimplementedPublicNodeServer
	node *Node
}

// func (server publicNodeServer) GetNodeStatus(ctx context.Context, statusRequest *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
// 	return &pb.NodeInfoResponse{
// 		Address: server.node.info.Address.String(),
// 		Name:    server.node.name,
// 		Balance: server.node.state.Manifest[server.node.info.Address].Balance,
// 		Hash:    server.node.state.LatestBlockHash().Hex(),
// 		Number:  server.node.state.LatestBlock().Header.Number,
// 	}, nil
// }

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
func (server publicNodeServer) AddTransaction(
	ctx context.Context, addPendingTransactionRequest *pb.AddPendingPublishCIDTransactionRequest) (
	*pb.AddPendingPublishCIDTransactionResponse, error) {
	nonce := server.node.state.PendingAccount2Nonce[server.node.info.Address] + 1
	tx := state.NewTx(
		server.node.info.Address,
		state.NewAddress(addPendingTransactionRequest.ToAddress),
		state.NewCID(
			addPendingTransactionRequest.Cid,
			addPendingTransactionRequest.Gateway,
			addPendingTransactionRequest.Name,
		),
		nonce, 0, state.TX_TYPE_001,
	)
	signedTx, err := wallet.SignTxWithKeystoreAccount(
		tx, server.node.info.Address, "test", wallet.GetKeystoreDirPath(server.node.datadir))
	if err != nil {
		return nil, err
	}
	server.node.AddPendingTX(signedTx)
	return &pb.AddPendingPublishCIDTransactionResponse{}, nil
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

func newNodeServer(n *Node) publicNodeServer {
	nodeServer := publicNodeServer{node: n}
	return nodeServer
}
