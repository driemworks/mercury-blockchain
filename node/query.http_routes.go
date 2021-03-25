package node

import "net/http"

//
func blockchainQueryHandler(w http.ResponseWriter, r *http.Request, node *Node) {

	writeRes(w, nil)
}
