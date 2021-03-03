package node

import (
	"fmt"
	"ftp2p/manifest"
	"ftp2p/wallet"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	goCid "github.com/ipfs/go-cid"
)

type ListInboxRequest struct {
	Limit int `json:"limit"`
}

type ListSentRequest struct {
}

type listInboxResponse struct {
	Inbox []manifest.InboxItem `json:"inbox"`
}

type listSentResponse struct {
	Sent []manifest.SentItem `json:"sent"`
}

// '/inbox'
func inboxHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	fmt.Println(r.URL.Query()["limit"])
	from := node.info.Address
	writeRes(w, listInboxResponse{node.state.Manifest[from].Inbox})
}

// '/sent'
func sentHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	from := node.info.Address
	writeRes(w, listSentResponse{node.state.Manifest[from].Sent})
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
	// needed?
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
