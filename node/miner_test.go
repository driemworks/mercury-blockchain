package node

import (
	"context"
	"encoding/hex"
	"ftp2p/manifest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestValidBlockHash(t *testing.T) {
	hexHash := "000000a293498234821349823482349823dffa"
	var hash = manifest.Hash{}
	hex.Decode(hash[:], []byte(hexHash))
	isValid := manifest.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' with 6 zeroes should be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "999999999"
	var hash = manifest.Hash{}
	hex.Decode(hash[:], []byte(hexHash))
	isValid := manifest.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' should not be valid", hexHash)
	}
}

func TestMine(t *testing.T) {
	miner := manifest.NewAddress("tony")
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
	if !manifest.IsBlockHashValid(mineBlockHash) {
		t.Fatal()
	}
	if minedBlock.Header.Miner != miner {
		t.Fatal("mined block's miner should match the pending block miner")
	}
}

func TestMineWithTimeout(t *testing.T) {
	miner := manifest.NewAddress("andrej")
	pendingBlock := createRandomPendingBlock(miner)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(miner common.Address) PendingBlock {
	return NewPendingBlock(manifest.Hash{}, 0, miner, []manifest.SignedTx{
		{manifest.Tx{From: manifest.NewAddress("tony"), To: manifest.NewAddress("theo"), CID: manifest.NewCID("", "")}, []byte{}},
	},
	)
}
