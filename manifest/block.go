package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

type Block struct {
	Header BlockHeader
	TXs    []Tx
}

type BlockHeader struct {
	Parent Hash   `json:"parent"`
	Time   uint64 `json:"time"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

/*
* Hash the block's transactions
 */
func (b *Block) Hash() (Hash, error) {
	txJson, err := json.Marshal(b.TXs)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

func NewBlock(parent Hash, time uint64, txs []Tx) Block {
	return Block{BlockHeader{parent, time}, txs}
}
