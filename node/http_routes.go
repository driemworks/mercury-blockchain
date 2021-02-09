package node

import (
	"driemcoin/main/manifest"
	"fmt"
	"net/http"
	"strconv"
	"time"
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
	Hash     manifest.Hash                          `json:"hash"`
	Manifest map[manifest.Account]manifest.Manifest `json:"manifest"`
}

type CidAddRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Cid  string `json:"cid"`
}

type CidAddResponse struct {
	Hash manifest.Hash `json:block_hash`
}

type StatusResponse struct {
	Hash       manifest.Hash `json:"block_hash"`
	Number     uint64        `json:"block_number"`
	knownPeers []PeerNode    `json:"known_peers`
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
func addCIDHandler(w http.ResponseWriter, r *http.Request, state *manifest.State) {
	req := CidAddRequest{}
	err := readRequest(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	cidTx := manifest.NewTx(manifest.NewAccount(req.From),
		manifest.NewAccount(req.To), manifest.NewCID(req.Cid), "")
	next := state.NextBlockNumber()
	block := manifest.NewBlock(state.LatestBlockHash(), uint64(time.Now().Unix()),
		next, []manifest.Tx{cidTx})
	hash, err := state.AddBlock(block)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, CidAddResponse{hash})
}

/**
*
 */
func nodeStatusHandler(w http.ResponseWriter, r *http.Request, state *manifest.State) {
	res := StatusResponse{
		Hash:   state.LatestBlockHash(),
		Number: state.LatestBlock().Header.Number,
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

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, true)

	node.AddPeer(peer)

	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())

	writeRes(w, AddPeerRes{true, ""})
}
