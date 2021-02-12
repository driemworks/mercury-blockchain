package node

import (
	"fmt"
	"ftp2p/main/manifest"
	"ftp2p/main/wallet"
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
	Hash       manifest.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	Alias      string              `json:"alias"`
	KnownPeers map[string]PeerNode `json:"known_peers"`
	PendingTxs []manifest.SignedTx `json:"pending_txs"`
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
	Alias   string            `json:"alias"`
	Mailbox manifest.Manifest `json:"mailbox"`
}

type cidAddRequest struct {
	To      string `json:"to"`
	Cid     string `json:"cid"`
	FromPwd string `json:"from_pwd"`
}

type cidAddResponse struct {
	Success bool `json:"success"`
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
		from, node.alias, node.state.Manifest[from]})
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
	nonce := node.state.Account2Nonce[node.info.Address] + 1
	// TODO - the cost to send a cid is always 1?
	// should this really go to the tx's to value, or to the 'system' (bootstrap) node?
	tx := manifest.NewTx(from, manifest.NewAddress(req.To), manifest.NewCID(req.Cid), nonce, 1)
	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, node.info.Address, req.FromPwd, wallet.GetKeystoreDirPath(node.datadir))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, cidAddResponse{Success: true})
}

// host:port/tokens POST
func requestTokensHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := sendTokensRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	if req.Amount <= 0 {
		writeErrRes(w, fmt.Errorf("Requested amount should be greater than zero but was %d", req.Amount))
		return
	}
	from := node.info.Address
	nonce := node.state.Account2Nonce[node.info.Address] + 1
	tx := manifest.NewTx(from, manifest.NewAddress(req.To), manifest.NewCID(""), nonce, float32(req.Amount))
	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, req.FromPwd,
		wallet.GetKeystoreDirPath(node.datadir))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, TokenRequestResponse{Success: true, Amount: req.Amount})
}

/**
*
 */
func nodeStatusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	res := StatusResponse{
		Hash:       node.state.LatestBlockHash(),
		Number:     node.state.LatestBlock().Header.Number,
		Alias:      node.alias,
		KnownPeers: node.knownPeers,
		PendingTxs: node.getPendingTXsAsArray(),
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
	minerRaw := r.URL.Query().Get("miner")

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}
	peer := NewPeerNode(peerIP, peerPort, false, manifest.NewAddress(minerRaw), true)
	node.AddPeer(peer)
	fmt.Printf("Peer "+rainbow.Green("'%s'")+" was added into KnownPeers\n", peer.TcpAddress())
	writeRes(w, AddPeerRes{true, ""})
}
