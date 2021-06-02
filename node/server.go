package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/driemworks/mercury-blockchain/core"
	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/driemworks/mercury-blockchain/wallet"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type nodeServer struct {
	pb.UnimplementedNodeServiceServer
	node *Node
}

func (server nodeServer) GetNodeStatus(
	ctx context.Context, statusRequest *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	nodeState := server.node.state.Catalog[server.node.miner]
	var channels []string
	for _, bytes := range nodeState.OwnedChannels {
		channels = append(channels, string(bytes))
	}
	return &pb.NodeInfoResponse{
		Address:  server.node.miner.Hex(),
		Balance:  nodeState.Balance,
		Channels: channels,
	}, nil
}

func (server nodeServer) Subscribe(
	joinChannelRequest *pb.JoinChannelRequest, stream pb.NodeService_SubscribeServer) error {
	// TODO verify provided tx hash -> later... for now assume it exists
	dataChan := make(chan core.MessageTransport)
	server.node.state.Subscriptions[joinChannelRequest.TxHash] = dataChan
	server.node.Join(context.Background(), joinChannelRequest.TxHash, 128,
		func(data *pubsub.Message) {
			d := fmt.Sprintf("%s", data.Data)
			pid, err := peer.IDFromBytes(data.From)
			if err != nil {
				log.Fatalln(err)
			}
			stream.Send(&pb.ChannelData{
				Data: d, From: pid.Pretty(), Topic: data.GetTopic(),
			})
		}, dataChan)
	select {}
}

func (server nodeServer) Publish(
	ctx context.Context, publishRequest *pb.PublishRequest) (*pb.PublishResponse, error) {
	if dataChan := server.node.state.Subscriptions[publishRequest.TxHash]; dataChan != nil {
		dataChan <- core.MessageTransport{[]byte(publishRequest.Message)}
	} else {
		fmt.Println("You must first be subscribed to the topic")
	}
	return &pb.PublishResponse{}, nil
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
func (server nodeServer) ListBlocks(
	listBlocksRequest *pb.ListBlocksRequest, stream pb.NodeService_ListBlocksServer) error {
	hash := state.Hash{}
	err := hash.UnmarshalText([]byte(listBlocksRequest.FromBlock))
	if err != nil {
		return err
	}
	blocks, err := state.GetBlocksAfter(hash, server.node.datadir)
	if err != nil {
		return err
	}

	for _, block := range blocks {
		blockHeader := pb.BlockHeaderMessage{}
		txs := make([]*pb.TransactionMessage, 0)
		for _, t := range block.TXs {
			hash, err := t.Hash()
			if err != nil {
				log.Fatalln("failed to hash the tx: ", err)
			}
			txMessage := pb.TransactionMessage{
				Author: t.Author.Hex(),
				Topic:  t.Topic,
				Hash:   hash.Hex(),
				// Nonce:  string(t.Nonce),
				// Time:   string(t.Time),
				// Signature: string(t.Sig),
			}
			// txs = append(txs, pb.TransactionMessage{
			// 	t.Author, t.Topic, t.Nonce, t.Time, t.Signature,
			// })
			txs = append(txs, &txMessage)
		}
		stream.Send(&pb.BlockResponse{
			BlockHeader: &blockHeader,
			Txs:         txs,
		})
	}
	return nil
}

/*
	Read/Write pending transactions
*/
func (server nodeServer) AddTransaction(
	ctx context.Context, addPendingTransactionRequest *pb.AddPendingTransactionRequest) (
	*pb.AddPendingTransactionResponse, error) {
	nonce := server.node.state.PendingAccount2Nonce[server.node.miner] + 1
	tx := state.NewTx(
		server.node.miner, addPendingTransactionRequest.Label, nonce,
	)
	signedTx, err := wallet.SignTxWithKeystoreAccount(
		tx, server.node.miner, addPendingTransactionRequest.Password,
		wallet.GetKeystoreDirPath(server.node.datadir))
	if err != nil {
		return nil, err
	}
	server.node.AddPendingTX(signedTx)
	txBytes, err := json.Marshal(signedTx)
	if err != nil {
		return nil, err
	}
	server.node.newPendingTXs <- core.MessageTransport{txBytes}
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
