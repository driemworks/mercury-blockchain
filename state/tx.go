package state

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewAddress(value string) common.Address {
	return common.HexToAddress(value)
}

type Tx struct {
	Author common.Address `json:"author"`
	Topic  string         `json:"topic"`
	Nonce  uint           `json:"nonce"`
	Time   uint64         `json:"time"`
}

type SignedTx struct {
	Tx
	Sig []byte `json:"signature"`
}

func NewTx(from common.Address, topic string, nonce uint) Tx {
	return Tx{from, topic, nonce, uint64(time.Now().Unix())}
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

	return recoveredAccount.Hex() == t.Author.Hex(), nil
}
