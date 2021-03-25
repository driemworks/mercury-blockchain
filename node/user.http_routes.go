package node

import (
	"fmt"
	"ftp2p/state"
	"ftp2p/wallet"
	"math"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	goCid "github.com/ipfs/go-cid"
)

type listInboxResponse struct {
	Inbox []state.InboxItem `json:"inbox"`
}

type listSentResponse struct {
	Sent []state.SentItem `json:"sent"`
}

type listInfoResponse struct {
	Address string  `json:"address"`
	Name    string  `json:"name"`
	Balance float32 `json:"balance"`
}

type peerResponse struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      uint64 `json:"port"`
	PublicKey string `json:"public_key"`
}

type knownPeersResponse struct {
	KnownPeers []peerResponse `json:"known_peers"`
}

type trustedPeersResponse struct {
	TrustedPeers []peerResponse `json:"trusted_peers"`
}

// '/inbox?from={}&limit={}'
/**
 */
func inboxHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	fromNode := r.URL.Query().Get("from")
	limitString := r.URL.Query().Get("limit")
	limit := -1
	if limitString != "" {
		limit, _ = strconv.Atoi(limitString)
	}
	thisNode := node.info.Address

	inbox := node.state.Manifest[thisNode].Inbox
	inboxItems := make([]state.InboxItem, 0)
	if fromNode != "" {
		for _, item := range inbox {
			if item.From.Hex() == fromNode {
				inboxItems = append(inboxItems, item)
			}
		}
	} else {
		inboxItems = inbox
	}

	if limit > -1 {
		inboxItems = inboxItems[0:int(math.Min(float64(limit), float64(len(inboxItems))))]
	}
	writeRes(w, listInboxResponse{inboxItems})
}

// '/sent?from={]&limit={}'
/**
 */
func sentHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	fromNode := r.URL.Query().Get("from")
	limitString := r.URL.Query().Get("limit")
	limit := -1
	if limitString != "" {
		limit, _ = strconv.Atoi(limitString)
	}
	thisNode := node.info.Address

	sent := node.state.Manifest[thisNode].Sent
	sentItems := make([]state.SentItem, 0)
	if fromNode != "" {
		for _, item := range sent {
			if item.To.Hex() == fromNode {
				sentItems = append(sentItems, item)
			}
		}
	} else {
		sentItems = sent
	}

	if limit > -1 {
		sentItems = sentItems[0:int(math.Min(float64(limit), float64(len(sentItems))))]
	}
	writeRes(w, listSentResponse{sentItems})
}

// '/info'
/**
 */
func infoHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	from := node.info.Address
	writeRes(w, listInfoResponse{from.Hex(), node.name,
		node.state.Manifest[from].Balance})
}

/**
 */
func knownPeersHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	knownPeers := make([]peerResponse, 0, len(node.knownPeers))
	for _, n := range node.knownPeers {
		knownPeers = append(knownPeers, peerResponse{n.Address.Hex(), n.Name, n.IP, n.Port, n.EncryptionPublicKey})
	}
	writeRes(w, knownPeersResponse{knownPeers})
}

/**
 */
func trustedPeersHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	trustedPeers := make([]peerResponse, 0, len(node.trustedPeers))
	for _, n := range node.trustedPeers {
		trustedPeers = append(trustedPeers, peerResponse{n.Address.Hex(), n.Name, n.IP, n.Port, n.EncryptionPublicKey})
	}
	writeRes(w, trustedPeersResponse{trustedPeers})
}

func sendCIDHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := cidAddRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	setupResponse(&w, r)
	// needed?
	if (*r).Method == "OPTIONS" {
		return
	}
	// safe to assume 'from' is a valid address
	from := node.info.Address
	to := state.NewAddress(req.To)
	// validations to make sure the recipient is valid addresses
	// if the 'to' parameter is empty, it will be assumed that anybody can access this data
	// i.e. it is public data for any node in the network
	if to.String() == common.HexToAddress("").String() && to.String() != "" {
		writeErrRes(w, fmt.Errorf("%s is an invalid 'to' address", to.String()))
		return
	}
	// TODO tx cost no FTC for now
	// check that the pending balance is greater than zero
	// if node.state.Manifest[from].Balance <= float32(0) {
	// 	writeErrRes(w, fmt.Errorf("your pending balance is non-positive. Please add funds and try again"))
	// 	return
	// }
	// verify  the tx contains a valid CID
	_, err = goCid.Decode(fmt.Sprintf("%s", req.Cid))
	if err != nil {
		writeErrRes(w, err)
		return
	}
	nonce := node.state.PendingAccount2Nonce[node.info.Address] + 1
	// TODO - the cost to send a cid is always 1?
	// should this really go to the tx's to value, or to the 'system' (bootstrap) node?
	tx := state.NewTx(from, state.NewAddress(req.To), state.NewCID(req.Cid, req.Gateway), nonce, 0)
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
	if node.knownPeers[req.TcpAddress].Address == state.NewAddress("0x0000000000000000000000000000000000000000") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error_code\": \"ERR_001\", \"error_desc\": \"no known node with provided address\"}"))
	}
	node.trustedPeers[req.TcpAddress] = node.knownPeers[req.TcpAddress]
	// TODO write trusted peers to a file.. perhaps as a transaction
	writeRes(w, struct{ trustedPeers map[string]PeerNode }{
		node.trustedPeers,
	})
}

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
		writeErrRes(w, fmt.Errorf("requested amount should be greater than zero but was %x", req.Amount))
		return
	}
	from := node.info.Address
	nonce := node.state.PendingAccount2Nonce[node.info.Address] + 1
	tx := state.NewTx(from, state.NewAddress(req.To), state.NewCID("", ""), nonce, float32(req.Amount))
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
