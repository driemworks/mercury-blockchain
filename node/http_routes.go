package node

import (
	"driemcoin/main/manifest"
	"driemcoin/main/wallet"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	goCid "github.com/ipfs/go-cid"
)

type SyncRes struct {
	Blocks []manifest.Block `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ManifestResponse struct {
	Hash     manifest.Hash                        `json:"hash"`
	Manifest map[common.Address]manifest.Manifest `json:"manifest"`
}

type CidAddRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Cid     string `json:"cid"`
	FromPwd string `json:"from_pwd"`
}

type CidAddResponse struct {
	Success bool `json:"success"`
}

type StatusResponse struct {
	Hash       manifest.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	Alias      string              `json:"alias"`
	KnownPeers map[string]PeerNode `json:"known_peers"`
	PendingTxs []manifest.SignedTx `json:"pending_txs"`
}

/**
* List the manifest
 */
func listManifestHandler(w http.ResponseWriter, r *http.Request, state *manifest.State) {
	writeRes(w, ManifestResponse{state.LatestBlockHash(), state.Manifest})
}

/**
*
 */
func addCIDHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := CidAddRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	from := manifest.NewAddress(req.From)
	// validations to make sure the sender/receiver are valid addresses
	if from.String() == common.HexToAddress("").String() {
		writeErrRes(w, fmt.Errorf("%s is an invalid 'from' sender", from.String()))
		return
	}
	// validations to make sure the CID is valid
	if req.FromPwd == "" {
		writeErrRes(w, fmt.Errorf("password to decrypt the %s address is required. 'from_pwd' is empty", from.String()))
		return
	}

	// verify  the tx contains a valid CID
	_, err = goCid.Decode(fmt.Sprintf("%s", req.Cid))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	nonce := node.state.GetNextAccountNonce(from)
	tx := manifest.NewTx(manifest.NewAddress(req.From),
		manifest.NewAddress(req.To), manifest.NewCID(req.Cid), nonce, "")
	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, req.FromPwd, wallet.GetKeystoreDirPath(node.datadir))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, CidAddResponse{Success: true})

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
	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())
	writeRes(w, AddPeerRes{true, ""})
}
