package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
)

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
			logrus.Infoln("Mining cancelled!")
			return state.Block{}, fmt.Errorf(rainbow.Red("mining cancelled. %s"), ctx.Err())
		default:
		}

		attempt++
		nonce = generateNonce()

		if attempt%1000000 == 0 || attempt == 1 {
			logrus.Infoln("Mining " + rainbow.Magenta(fmt.Sprintf("%d", len(pb.txs))) + " Pending TXs. Attempt: " + rainbow.Magenta(fmt.Sprintf("%d", attempt)))
		}

		block = state.NewBlock(pb.parent, pb.time, pb.number, pb.txs, nonce, pb.miner, attempt)
		blockHash, err := block.Hash()
		if err != nil {
			return state.Block{}, fmt.Errorf("couldn't mine block. %s", err.Error())
		}

		hash = blockHash
	}

	logrus.Infof("\nMined new Block '%v':\n", info(fmt.Sprint(hash)))
	logrus.Infof("\tHeight: '%v'\n", info(fmt.Sprint(block.Header.Number)))
	logrus.Infof("\tNonce: '%v'\n", info(fmt.Sprint(block.Header.Nonce)))
	logrus.Infof("\tCreated: '%v'\n", info(fmt.Sprint(block.Header.Time)))
	logrus.Infof("\tMiner: '%v'\n", info(fmt.Sprint(block.Header.Miner)))
	logrus.Infof("\tParent: '%v'\n\n", info(fmt.Sprint(block.Header.Parent.Hex())))
	logrus.Infof("\tAttempt: '%v'\n", info(fmt.Sprint(attempt)))
	logrus.Infof("\tTime: %s\n\n", info(fmt.Sprint(time.Since(start))))

	return block, nil
}

func info(msg string) string {
	return rainbow.Magenta(msg)
}
