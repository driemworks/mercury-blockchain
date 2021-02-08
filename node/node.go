package node

import (
	"driemcoin/main/manifest"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const httpPort = 8080

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
	Hash   manifest.Hash `json:"block_hash"`
	Number uint64        `json:"block_number"`
}

/**
* Start the node's HTTP client
 */
func Run(datadir string) error {
	fmt.Println(fmt.Sprintf("Starting node HTTP client on port %d", httpPort))
	state, err := manifest.NewStateFromDisk(datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	// list manifest
	http.HandleFunc("/manifest/list", func(w http.ResponseWriter, r *http.Request) {
		listManifestHandler(w, r, state)
	})
	// send CID to someone
	http.HandleFunc("/cid/add", func(w http.ResponseWriter, r *http.Request) {
		addCIDHandler(w, r, state)
	})
	// get the nodes' status
	// http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request)) {

	// }
	return http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
}

// func nodeStatusHandler(w http.ResponseWriter, r *http.Request, state *manifest.State) {
// 	res := StatusResponse{
// 		Hash: state.LatestBlockHash(),
// 		Number: state.LatestBlock().Header.Number,
// 	}
// }

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
	// TODO - consider renaming this function
	cidTx := manifest.NewTx(manifest.NewAccount(req.From),
		manifest.NewAccount(req.To), manifest.NewCID(req.Cid), "")
	err = state.AddTx(cidTx)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	hash, err := state.Persist()
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, CidAddResponse{hash})
}

/**
*
 */
func readRequest(r *http.Request, reqBody interface{}) error {
	reqBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("Unable to umarshal request body %s", err.Error())
	}
	defer r.Body.Close()
	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("Unable to umarshal request body %s", err.Error())
	}
	return nil
}

/**
*
 */
func writeErrRes(w http.ResponseWriter, err error) {
	jsonErrRes, _ := json.Marshal(ErrorResponse{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonErrRes)
}

/**
*
 */
func writeRes(w http.ResponseWriter, content interface{}) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(contentJson)
}
