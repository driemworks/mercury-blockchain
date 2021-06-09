package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/raphamorim/go-rainbow"
	"github.com/sirupsen/logrus"
)

func generateNonce() uint32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Uint32()
}

func ElectBlockCreator() (*common.Address, error) {
	return nil, nil
}

func Mine(ctx context.Context, pb state.PendingBlock) (state.Block, error) {
	if len(pb.Txs) == 0 {
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
			logrus.Infoln("Mining " + rainbow.Magenta(fmt.Sprintf("%d", len(pb.Txs))) + " Pending TXs. Attempt: " + rainbow.Magenta(fmt.Sprintf("%d", attempt)))
		}

		block = state.NewBlock(pb.Parent, pb.Time, pb.Number, pb.Txs, nonce, pb.Miner, attempt)
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
