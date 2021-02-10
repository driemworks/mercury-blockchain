package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.Hex()), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return bytes.Equal(emptyHash[:], h[:])
}

type Block struct {
	Header BlockHeader
	TXs    []SignedTx
}

type BlockHeader struct {
	Parent Hash           `json:"parent"`
	Time   uint64         `json:"time"`
	Number uint64         `json:"number"`
	Nonce  uint32         `json:"nonce"`
	Miner  common.Address `json:"miner"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

/*
 Hash the block's transactions
*/
func (b *Block) Hash() (Hash, error) {
	txJson, err := json.Marshal(b.TXs)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

/*
	NewBlock
*/
func NewBlock(parent Hash, time uint64, number uint64, txs []SignedTx, nonce uint32, miner common.Address) Block {
	return Block{BlockHeader{parent, time, number, nonce, miner}, txs}
}

/*
	IsBlockHashValid
*/
func IsBlockHashValid(hash Hash) bool {
	// for now, check that first 4 values are all '0'
	return fmt.Sprintf("%x", hash[0]) == "0" &&
		fmt.Sprintf("%x", hash[1]) == "0" &&
		fmt.Sprintf("%x", hash[2]) == "0" &&
		fmt.Sprintf("%x", hash[3]) != "0"

}
