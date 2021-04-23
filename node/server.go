package node

import (
	"context"
	"encoding/json"

	"github.com/driemworks/mercury-blockchain/core"
	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/driemworks/mercury-blockchain/wallet"
)

type publicNodeServer struct {
	pb.UnimplementedPublicNodeServer
	node *Node
}

// // StatusResponse TODO
// type StatusResponse struct {
// 	Hash         state.Hash               `json:"block_hash"`
// 	Number       uint64                   `json:"block_number"`
// 	Alias        string                   `json:"alias"`
// 	KnownPeers   map[string]core.PeerNode `json:"known_peers"`
// 	TrustedPeers map[string]core.PeerNode `json:"trusted_peers"`
// 	PendingTxs   []state.SignedTx         `json:"pending_txs"`
// }

func (server publicNodeServer) GetNodeStatus(ctx context.Context, statusRequest *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	return &pb.NodeInfoResponse{
		Address: server.node.info.Address.String(),
		Name:    server.node.name,
		Balance: server.node.state.Manifest[server.node.info.Address].Balance,
		Hash:    server.node.state.LatestBlockHash().Hex(),
		Number:  server.node.state.LatestBlock().Header.Number,
	}, nil
}

/*
	Read/Write known peers
*/
func (server publicNodeServer) ListKnownPeers(listKnownPeersRequest *pb.ListKnownPeersRequest,
	stream pb.PublicNode_ListKnownPeersServer) error {
	for tcp, pn := range server.node.knownPeers {
		if err := stream.Send(&pb.ListKnownPeersResponse{
			Name:        pn.Name,
			Ip:          server.node.knownPeers[tcp].IP,
			Port:        server.node.knownPeers[tcp].Port,
			IsBootstrap: pn.IsBootstrap,
			Address:     pn.Address.String(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// TODO: Need to add authentication to this... but this is fine for now
func (server publicNodeServer) JoinKnownPeers(ctx context.Context, joinKnownPeersRequest *pb.JoinKnownPeersRequest) (*pb.JoinKnownPeersResponse, error) {
	// TODO consider moving new address function into core package
	server.node.knownPeers[joinKnownPeersRequest.Address] = core.NewPeerNode(
		joinKnownPeersRequest.Name, joinKnownPeersRequest.Ip, joinKnownPeersRequest.Port,
		joinKnownPeersRequest.IsBootstrap, state.NewAddress(joinKnownPeersRequest.Address), true,
	)
	return &pb.JoinKnownPeersResponse{}, nil
}

/*
	Read
*/
func (server publicNodeServer) ListBlocks(listBlocksRequest *pb.ListBlocksRequest,
	stream pb.PublicNode_ListBlocksServer) error {
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
		encoded, err := json.Marshal(block)
		if err != nil {
			break
		}
		stream.Send(&pb.BlockResponse{
			Block: encoded,
		})
	}
	return nil
}

/*
	Read/Write pending transactions
*/
func (server publicNodeServer) AddPendingPublishCIDTransaction(
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
		tx, server.node.info.Address, server.node.password, wallet.GetKeystoreDirPath(server.node.datadir))
	if err != nil {
		return nil, err
	}
	server.node.AddPendingTX(signedTx)
	return &pb.AddPendingPublishCIDTransactionResponse{}, nil
}

// TODO for now this is only the publish cid tx... will generalize later
func (server publicNodeServer) ListPendingTransactions(request *pb.ListPendingTransactionsRequest,
	stream pb.PublicNode_ListPendingTransactionsServer) error {
	for _, tx := range server.node.pendingTXs {
		// TODO => can't always assume this... need to add a tx type?
		bytes, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		err = stream.Send(&pb.PendingTransactionResponse{
			// Nonce:       uint32(tx.Nonce),
			// Time:        tx.Time,
			// Cid:         cid.CID,
			// Gateway:     cid.IPFSGateway,
			// Name:        cid.Name,
			// FromAddress: tx.From.String(),
			// ToAddress:   tx.To.String(),
			SignedTx: bytes,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func newNodeServer(n *Node) publicNodeServer {
	nodeServer := publicNodeServer{node: n}
	return nodeServer
}
