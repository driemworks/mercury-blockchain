package node

import (
	"fmt"
	"ftp2p/manifest"
	"ftp2p/wallet"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	goCid "github.com/ipfs/go-cid"
	"github.com/raphamorim/go-rainbow"
)

// SyncRes is the response struct for representing new blocks to sync
type SyncRes struct {
	Blocks []manifest.Block `json:"blocks"`
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
	Hash         manifest.Hash       `json:"block_hash"`
	Number       uint64              `json:"block_number"`
	Alias        string              `json:"alias"`
	KnownPeers   map[string]PeerNode `json:"known_peers"`
	TrustedPeers map[string]PeerNode `json:"trusted_peers"`
	PendingTxs   []manifest.SignedTx `json:"pending_txs"`
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
	Hash     manifest.Hash                        `json:"hash"`
	Manifest map[common.Address]manifest.Manifest `json:"manifest"`
}

type userMailboxResponse struct {
	Hash    manifest.Hash     `json:"hash"`
	Address common.Address    `json:"address"`
	Name    string            `json:"name"`
	Mailbox manifest.Manifest `json:"mailbox"`
}

type cidAddRequest struct {
	To      string `json:"to"`
	Cid     string `json:"cid"`
	Gateway string `json:"gateway"`
	FromPwd string `json:"from_pwd"`
}

type encryptDataRequest struct {
	Data    string `json:"data"` // doing this for now, change to multipart upload later
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
* List the manifest
 */
func viewMailboxHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	// verify that addressParam is a valid address
	from := node.info.Address
	// validations to make sure the sender/receiver are valid addresses
	if from.String() == common.HexToAddress("").String() {
		writeErrRes(w, fmt.Errorf("%s is not a valid address", from))
		return
	}
	// address := manifest.NewAddress(from)
	writeRes(w, userMailboxResponse{node.state.LatestBlockHash(),
		from, node.name, node.state.Manifest[from]})
}

/**
* TODO - can probably expand this to handle generic state mutation, then restrict the
 use of it based on endpoint parameters
*/
func addCIDHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := cidAddRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	setupResponse(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}
	// safe to assume 'from' is a valid address
	from := node.info.Address
	to := manifest.NewAddress(req.To)
	// validations to make sure the recipient is valid addresses
	if to.String() == common.HexToAddress("").String() {
		writeErrRes(w, fmt.Errorf("%s is an invalid 'to' address", to.String()))
		return
	}
	if req.FromPwd == "" {
		writeErrRes(w, fmt.Errorf("password to decrypt the %s address is required. 'from_pwd' is empty", from))
		return
	}
	// check that the pending balance is greater than zero
	if node.state.Manifest[from].Balance <= float32(0) {
		writeErrRes(w, fmt.Errorf("your pending balance is non-positive. Please add funds and try again"))
		return
	}
	// verify  the tx contains a valid CID
	_, err = goCid.Decode(fmt.Sprintf("%s", req.Cid))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	nonce := node.state.PendingAccount2Nonce[node.info.Address] + 1
	// TODO - the cost to send a cid is always 1?
	// should this really go to the tx's to value, or to the 'system' (bootstrap) node?
	tx := manifest.NewTx(from, manifest.NewAddress(req.To), manifest.NewCID(req.Cid, req.Gateway), nonce, 1)
	signedTx, err := wallet.SignTxWithKeystoreAccount(
		tx, node.info.Address, req.FromPwd, wallet.GetKeystoreDirPath(node.datadir))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	err = node.AddPendingTX(signedTx)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, struct{}{})
}

func addTrustedPeerNodeHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := AddTrustedPeerNodeRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	if node.knownPeers[req.TcpAddress].Address == manifest.NewAddress("0x0000000000000000000000000000000000000000") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error_code\": \"ERR_001\", \"error_desc\": \"no known node with provided address\"}"))
	}
	node.trustedPeers[req.TcpAddress] = node.knownPeers[req.TcpAddress]
	writeRes(w, struct{ trustedPeers map[string]PeerNode }{
		node.trustedPeers,
	})
}

// host:port/tokens POST
func sendTokensHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := sendTokensRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	setupResponse(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}
	if req.Amount <= 0 {
		writeErrRes(w, fmt.Errorf("Requested amount should be greater than zero but was %d", req.Amount))
		return
	}
	from := node.info.Address
	nonce := node.state.PendingAccount2Nonce[node.info.Address] + 1
	tx := manifest.NewTx(from, manifest.NewAddress(req.To), manifest.NewCID("", ""), nonce, float32(req.Amount))
	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, req.FromPwd,
		wallet.GetKeystoreDirPath(node.datadir))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	err = node.AddPendingTX(signedTx)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, TokenRequestResponse{Success: true, Amount: req.Amount})
}

func encryptDataHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := encryptDataRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	// TODO - condense/cleanup these params
	encryptedData, err := wallet.Encrypt(
		wallet.GetKeystoreDirPath(node.datadir),
		req.FromPwd,
		node.info.Address,
		manifest.NewAddress(req.To).Hash().Bytes(),
		[]byte(req.Data),
		wallet.X25519,
	)
	// encryptedData, err := wallet.Encrypt(
	// 	manifest.NewAddress(req.To).Hash().Bytes(),
	// 	[]byte(req.Data),
	// 	wallet.X25519,
	// )
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, EncryptDataResponse{EncryptedData: encryptedData})
}

func decryptDataHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := DecryptDataRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	decryptedData, err := wallet.Decrypt(
		wallet.GetKeystoreDirPath(node.datadir),
		node.info.Address,
		req.FromPwd,
		&req.EncryptedData,
	)
	// prvKey, err := wallet.RecoverPrivateKey(
	// 	wallet.GetKeystoreDirPath(node.datadir),
	// 	req.FromPwd,
	// 	node.info.Address,
	// )
	// if err != nil {
	// 	writeErrRes(w, err)
	// 	return
	// }
	// decryptedData, err := wallet.Decrypt(
	// 	prvKey,
	// 	&req.EncryptedData,
	// )
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, DecryptDataResponse{string(decryptedData)})
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

	hash := manifest.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	blocks, err := manifest.GetBlocksAfter(hash, node.datadir)
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

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}
	peer := NewPeerNode(peerName, peerIP, peerPort, false, manifest.NewAddress(minerRaw), true)
	node.AddPeer(peer)
	fmt.Printf("Peer "+rainbow.Green("'%s'")+" was added into KnownPeers\n", peer.TcpAddress())
	writeRes(w, AddPeerRes{true, ""})
}
