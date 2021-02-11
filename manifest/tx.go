package manifest

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
	From   common.Address `json: "from"`
	To     common.Address `json: "to"`
	CID    CID            `json: "cid"`
	Nonce  uint           `json:"nonce"`
	Time   uint64         `json:"time"`
	Amount float32        `json:"amount"`
}

type SignedTx struct {
	Tx
	Sig []byte `json:"signature"`
}

func NewTx(from common.Address, to common.Address, cid CID, nonce uint, amount float32) Tx {
	return Tx{from, to, cid, nonce, uint64(time.Now().Unix()), amount}
}

func NewSignedTx(tx Tx, sig []byte) SignedTx {
	return SignedTx{tx, sig}
}

// func (t Tx) IsReward() bool {
// 	// what would be a meaningful reward in the context of file sharing?
// 	// I suppose... the "in-app" currency maybe?
// 	// maybe there could be different tiers? based on number of transactions/day
// 	return t.Data == "reward"
// }

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
