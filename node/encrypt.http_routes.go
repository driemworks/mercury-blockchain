package node

import (
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

	_myPubKey := [32]byte{}
	copy(_myPubKey[:], node.info.Address.Hash().Bytes()[:32])

	privateKey, err := wallet.RecoverPrivateKey(wallet.GetKeystoreDirPath(node.datadir),
		req.FromPwd, node.info.Address)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	_privateKey := [32]byte{}
	copy(_privateKey[:], privateKey[:32])

	recipientKey := []byte(req.To)
	_recipientKey := [32]byte{}
	copy(_recipientKey[:], recipientKey[:32])

	encryptedData, err := wallet.Encrypt(
		_recipientKey,
		[]byte(req.Data),
		wallet.X25519,
	)
	// _myPubKey,
	// _privateKey,
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
	privateKey, err := wallet.RecoverPrivateKey(wallet.GetKeystoreDirPath(node.datadir),
		req.FromPwd, node.info.Address)
	if err != nil {
		writeErrRes(w, err)
		return
	}
	_privateKey := [32]byte{}
	copy(_privateKey[:], privateKey[:32])

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
