package state

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/driemworks/mercury-blockchain/core"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// these constants directly correlate to user actions
const (
	TX_TYPE_001 = "PUBLISH"    // publish a cid to the network
	TX_TYPE_002 = "TRUST_PEER" // add a peer to the trusted peers
)

func NewAddress(value string) common.Address {
	return common.HexToAddress(value)
}

type Tx struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Payload CID            `json:"payload"`
	Nonce   uint           `json:"nonce"`
	Time    uint64         `json:"time"`
	Amount  float32        `json:"amount"`
}

// type TransactionPayload struct {
// 	// Value interface{}
// 	cid CID
// }

func NewTrustPeerTransactionPayload(pn core.PeerNode) TrustPeerTransactionPayload {
	return TrustPeerTransactionPayload{
		pn.Address, pn.IP, pn.IsBootstrap, pn.Name, pn.Port,
	}
}

// everything in PeerNode except the Connected field
type TrustPeerTransactionPayload struct {
	Address     common.Address `json:"address"`
	IP          string         `json:"ip"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Name        string         `json:"name"`
	Port        uint64         `json:"port"`
}

type SignedTx struct {
	Tx
	Sig []byte `json:"signature"`
}

func NewTx(from common.Address, to common.Address, payload CID, nonce uint, amount float32, txType string) Tx {
	return Tx{from, to, payload, nonce, uint64(time.Now().Unix()), amount}
}

func NewSignedTx(tx Tx, sig []byte) SignedTx {
	return SignedTx{tx, sig}
}

func (t Tx) Hash() (Hash, error) {
	txJson, err := json.Marshal(t)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

func (t Tx) Encode() ([]byte, error) {
	return json.Marshal(t)
}

func (t SignedTx) IsAuthentic() (bool, error) {
	txHash, err := t.Tx.Hash()
	if err != nil {
		return false, err
	}

	recoveredPubKey, err := crypto.SigToPub(txHash[:], t.Sig)
	if err != nil {
		return false, err
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	return recoveredAccount.Hex() == t.From.Hex(), nil
}
