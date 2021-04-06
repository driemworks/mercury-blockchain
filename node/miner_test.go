package node

import (
	"context"
	"encoding/hex"
	"ftp2p/state"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestValidBlockHash(t *testing.T) {
	hexHash := "000000a293498234821349823482349823dffa"
	var hash = state.Hash{}
	hex.Decode(hash[:], []byte(hexHash))
	isValid := state.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' with 6 zeroes should be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "999999999"
	var hash = state.Hash{}
	hex.Decode(hash[:], []byte(hexHash))
	isValid := state.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' should not be valid", hexHash)
	}
}

func TestMine(t *testing.T) {
	miner := state.NewAddress("tony")
	pendingBlock := createRandomPendingBlock(miner)
	ctx := context.Background()
	minedBlock, err := Mine(ctx, pendingBlock)
	if err != nil {
		t.Fatal(err)
	}
	mineBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlockHashValid(mineBlockHash) {
		t.Fatal()
	}
	if minedBlock.Header.Miner != miner {
		t.Fatal("mined block's miner should match the pending block miner")
	}
}

func TestMineWithTimeout(t *testing.T) {
	miner := state.NewAddress("andrej")
	pendingBlock := createRandomPendingBlock(miner)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(miner common.Address) PendingBlock {
	return NewPendingBlock(state.Hash{}, 0, miner, []state.SignedTx{
		{state.Tx{From: state.NewAddress("tony"), To: state.NewAddress("theo"),
			Payload: state.TransactionPayload{state.NewCID("", "")}}, []byte{}},
	},
	)
}
