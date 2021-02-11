package node

import (
	"context"
	"driemcoin/main/manifest"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// DefaultMiner is the miner address used if one is not provided
const DefaultMiner = "0xasdf"

// PendingBlock represents a block before it has been mined
type PendingBlock struct {
	parent manifest.Hash
	number uint64
	time   uint64
	miner  common.Address
	txs    []manifest.SignedTx
}

func NewPendingBlock(parent manifest.Hash, number uint64, miner common.Address, txs []manifest.SignedTx) PendingBlock {
	return PendingBlock{parent, number, uint64(time.Now().Unix()), miner, txs}
}

func generateNonce() uint32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Uint32()
}

func Mine(ctx context.Context, pb PendingBlock) (manifest.Block, error) {
	if len(pb.txs) == 0 {
		return manifest.Block{}, fmt.Errorf("block is empty - there is nothing to mine")
	}

	start := time.Now()
	attempt := 0
	var block manifest.Block
	var hash manifest.Hash
	var nonce uint32

	for !manifest.IsBlockHashValid(hash) {
		select {
		case <-ctx.Done():
			fmt.Println("Mining cancelled!")

			return manifest.Block{}, fmt.Errorf("mining cancelled. %s", ctx.Err())
		default:
		}

		attempt++
		nonce = generateNonce()

		if attempt%1000000 == 0 || attempt == 1 {
			fmt.Printf("Mining %d Pending TXs. Attempt: %d.\n", len(pb.txs), attempt)
		}

		block = manifest.NewBlock(pb.parent, pb.time, pb.number, pb.txs, nonce, pb.miner)
		blockHash, err := block.Hash()
		if err != nil {
			return manifest.Block{}, fmt.Errorf("couldn't mine block. %s", err.Error())
		}

		hash = blockHash
	}

	fmt.Printf("\nMined new Block '%x' using PoW🎉🎉🎉%s:\n", hash, manifest.Unicode("\\U1F389"))
	fmt.Printf("\tHeight: '%v'\n", block.Header.Number)
	fmt.Printf("\tNonce: '%v'\n", block.Header.Nonce)
	fmt.Printf("\tCreated: '%v'\n", block.Header.Time)
	fmt.Printf("\tMiner: '%v'\n", block.Header.Miner)
	fmt.Printf("\tParent: '%v'\n\n", block.Header.Parent.Hex())
	fmt.Printf("\tAttempt: '%v'\n", attempt)
	fmt.Printf("\tTime: %s\n\n", time.Since(start))

	return block, nil
}
