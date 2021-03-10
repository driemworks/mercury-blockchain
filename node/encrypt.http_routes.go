package node

import (
	"fmt"
	"ftp2p/wallet"
	"net/http"
)

func encryptDataHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := encryptDataRequest{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	// if req.To not provided, peer node is yourself
	var trustedPeerNode PeerNode
	if req.To == "" {
		trustedPeerNode = node.info
	} else {
		for pn := range node.trustedPeers {
			if node.trustedPeers[pn].Address.String() == req.To {
				trustedPeerNode = node.trustedPeers[pn]
			}
		}
	}

	var recipientKey [32]byte
	copy(recipientKey[:], []byte(trustedPeerNode.EncryptionPublicKey))
	if trustedPeerNode.IP == "" {
		writeErrRes(w, fmt.Errorf("node with address %s is not a trusted peer", req.To))
	}
	keys, err := wallet.LoadEncryptionKeys(node.datadir, req.FromPwd)
	var publicKey [32]byte
	copy(publicKey[:], keys[:32])

	var privateKey [32]byte
	copy(privateKey[:], keys[32:])

	if err != nil {
		writeErrRes(w, err)
		return
	}
	encryptedData, err := wallet.Encrypt(
		publicKey,
		privateKey,
		recipientKey,
		[]byte(req.Data),
		wallet.X25519,
	)
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
	keys, err := wallet.LoadEncryptionKeys(node.datadir, req.FromPwd)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	_privateKey := [32]byte{}
	copy(_privateKey[:], keys[32:])

	decryptedData, err := wallet.Decrypt(
		_privateKey,
		&req.EncryptedData,
	)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	writeRes(w, DecryptDataResponse{string(decryptedData)})
}
