package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/driemworks/mercury-blockchain/state"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

// DefaultMiner is the miner address used if one is not provided
const DefaultMiner = "0xasdf"

// PendingBlock represents a block before it has been mined
type PendingBlock struct {
	parent state.Hash
	number uint64
	time   uint64
	miner  common.Address
	txs    []state.SignedTx
}

func NewPendingBlock(parent state.Hash, number uint64, miner common.Address, txs []state.SignedTx) PendingBlock {
	return PendingBlock{parent, number, uint64(time.Now().Unix()), miner, txs}
}

func generateNonce() uint32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Uint32()
}

func Mine(ctx context.Context, pb PendingBlock) (state.Block, error) {
	if len(pb.txs) == 0 {
		return state.Block{}, fmt.Errorf(rainbow.Red("block is empty - there is nothing to mine"))
	}

	start := time.Now()
	attempt := 0
	var block state.Block
	var hash state.Hash
	var nonce uint32

	for !state.IsBlockHashValid(hash) {
		select {
		case <-ctx.Done():
			fmt.Println("Mining cancelled!")

			return state.Block{}, fmt.Errorf(rainbow.Red("mining cancelled. %s"), ctx.Err())
		default:
		}

		attempt++
		nonce = generateNonce()

		if attempt%1000000 == 0 || attempt == 1 {
			fmt.Printf("Mining "+rainbow.Magenta("%d")+" Pending TXs. Attempt: "+rainbow.Magenta("%d")+".\n", len(pb.txs), attempt)
		}

		block = state.NewBlock(pb.parent, pb.time, pb.number, pb.txs, nonce, pb.miner, attempt)
		blockHash, err := block.Hash()
		if err != nil {
			return state.Block{}, fmt.Errorf("couldn't mine block. %s", err.Error())
		}

		hash = blockHash
	}

	fmt.Printf("\nMined new Block '%v':\n", info(fmt.Sprint(hash)))
	fmt.Printf("\tHeight: '%v'\n", info(fmt.Sprint(block.Header.Number)))
	fmt.Printf("\tNonce: '%v'\n", info(fmt.Sprint(block.Header.Nonce)))
	fmt.Printf("\tCreated: '%v'\n", info(fmt.Sprint(block.Header.Time)))
	fmt.Printf("\tMiner: '%v'\n", info(fmt.Sprint(block.Header.Miner)))
	fmt.Printf("\tParent: '%v'\n\n", info(fmt.Sprint(block.Header.Parent.Hex())))
	fmt.Printf("\tAttempt: '%v'\n", info(fmt.Sprint(attempt)))
	fmt.Printf("\tTime: %s\n\n", info(fmt.Sprint(time.Since(start))))

	return block, nil
}

func info(msg string) string {
	return rainbow.Magenta(msg)
}
