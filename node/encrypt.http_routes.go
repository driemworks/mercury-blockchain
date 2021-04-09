package node

import (
	"fmt"
	com "ftp2p/common"
	"ftp2p/state"
	"ftp2p/wallet"
	"net/http"
)

func encryptDataHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := EncryptDataRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	// if req.To not provided, peer node is yourself
	var trustedPeerNode com.PeerNode
	if req.To == "" {
		trustedPeerNode = node.info
	} else {
		for pn := range node.trustedPeers {
			if node.trustedPeers[pn].Address.String() == req.To {
				trustedPeerNode = node.trustedPeers[pn]
			}
		}
	}
	if trustedPeerNode.IP == "" {
		writeErrRes(w, fmt.Errorf("node with address %s is not a trusted peer", req.To))
	} else {
		var receiverPublicKey [32]byte
		copy(receiverPublicKey[:], trustedPeerNode.Address.Hash().Bytes())
		encryptedData, err := wallet.Encrypt(
			receiverPublicKey,
			[]byte(req.Data),
			wallet.X25519,
		)
		if err != nil {
			writeErrRes(w, err)
			return
		}
		writeRes(w, EncryptDataResponse{EncryptedData: encryptedData})
	}
}

func decryptDataHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := DecryptDataRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	fmt.Println(req.EncryptedData)
	decryptedData, err := wallet.Decrypt(
		req.FromPwd,
		state.GetKeystoreDirPath(node.datadir),
		req.EncryptedData,
	)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, DecryptDataResponse{string(decryptedData)})
}
