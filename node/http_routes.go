package node

import (
	"fmt"
	com "ftp2p/common"
	"ftp2p/state"
	"ftp2p/wallet"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

// SyncRes is the response struct for representing new blocks to sync
type SyncRes struct {
	Blocks []state.Block `json:"blocks"`
}

// AddPeerRes is the response struct to represent if new peer addition
type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// ErrorResponse to represent any error that happens
type ErrorResponse struct {
	Error string `json:"error"`
}

// StatusResponse TODO
type StatusResponse struct {
	Hash         state.Hash              `json:"block_hash"`
	Number       uint64                  `json:"block_number"`
	Alias        string                  `json:"alias"`
	KnownPeers   map[string]com.PeerNode `json:"known_peers"`
	TrustedPeers map[string]com.PeerNode `json:"trusted_peers"`
	PendingTxs   []state.SignedTx        `json:"pending_txs"`
}

type TokenRequestResponse struct {
	Success bool    `json:"success"`
	Amount  float32 `json:"amount"`
}

type sendTokensRequest struct {
	Amount  float32 `json:"amount"`
	To      string  `json:"to"`
	FromPwd string  `json:"from_pwd"`
}

type manifestResponse struct {
	Hash     state.Hash                        `json:"hash"`
	Manifest map[common.Address]state.Manifest `json:"manifest"`
}

type userMailboxResponse struct {
	Hash    state.Hash     `json:"hash"`
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Mailbox state.Manifest `json:"mailbox"`
}

type cidAddRequest struct {
	To      string `json:"to"`
	Cid     string `json:"cid"`
	Gateway string `json:"gateway"`
	FromPwd string `json:"from_pwd"`
}

type encryptDataRequest struct {
	Data    string `json:"data"` // doing this for now, change to multipart upload later?
	To      string `json:"to"`
	FromPwd string `json:"from_pwd"`
}

type EncryptDataResponse struct {
	EncryptedData *wallet.EncryptedData `json:"encrypted_data"`
}

type DecryptDataRequest struct {
	EncryptedData wallet.EncryptedData `json:"encrypted_data"`
	FromPwd       string               `json:"from_pwd"`
}

type DecryptDataResponse struct {
	Data string `json:"data"`
}

type AddTrustedPeerNodeRequest struct {
	TcpAddress string `json:"tcp_address"`
}

/**
*
 */
func nodeStatusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	res := StatusResponse{
		Hash:         node.state.LatestBlockHash(),
		Number:       node.state.LatestBlock().Header.Number,
		Alias:        node.name,
		KnownPeers:   node.knownPeers,
		TrustedPeers: node.trustedPeers,
		PendingTxs:   node.getPendingTXsAsArray(),
	}
	writeRes(w, res)
}

/**
*
 */
func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	reqHash := r.URL.Query().Get("fromBlock")

	hash := state.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	blocks, err := state.GetBlocksAfter(hash, node.datadir)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, SyncRes{Blocks: blocks})
}

/**
*
 */
func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get("ip")
	peerPortRaw := r.URL.Query().Get("port")
	peerName := r.URL.Query().Get("name")
	minerRaw := r.URL.Query().Get("miner")
	encryptionPublicKey := r.URL.Query().Get("publicKey")

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}
	peer := com.NewPeerNode(peerName, peerIP, peerPort, false, state.NewAddress(minerRaw), encryptionPublicKey, true)
	node.AddPeer(peer)
	fmt.Printf("Peer "+rainbow.Green("'%s'")+" was added into KnownPeers\n", peer.TcpAddress())
	writeRes(w, AddPeerRes{true, ""})
}
