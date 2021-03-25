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
	PoW    int            `json:"proof_of_work"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

/*
 Hash the block's transactions
*/
func (b *Block) Hash() (Hash, error) {
	txJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

/*
	NewBlock
*/
func NewBlock(parent Hash, time uint64, number uint64, txs []SignedTx, nonce uint32, miner common.Address, pow int) Block {
	return Block{BlockHeader{parent, time, number, nonce, miner, pow}, txs}
}

/*
	IsBlockHashValid
*/
func IsBlockHashValid(hash Hash) bool {
	return fmt.Sprintf("%x", hash[0]) == "1" &&
		fmt.Sprintf("%x", hash[1]) == "0" &&
		fmt.Sprintf("%x", hash[2]) == "1" &&
		fmt.Sprintf("%x", hash[3]) != "0"

}
