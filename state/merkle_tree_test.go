package state

import (
	"crypto/sha256"
	"testing"
)

//TestSHA256Content implements the Content interface provided by merkletree and represents the content stored in the tree.
type TestSHA256Content struct {
	x string
}

//CalculateHash hashes the values of a TestSHA256Content
func (t TestSHA256Content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

//Equals tests for equality of two Contents
func (t TestSHA256Content) Equals(other Content) (bool, error) {
	return t.x == other.(TestSHA256Content).x, nil
}

func TestNewTree(t *testing.T) {
	test_tx := "hello tx world"
	txs := []Content{
		TestSHA256Content{x: "hello tx world"},
		TestSHA256Content{x: "test 1"},
		TestSHA256Content{x: "test 2"},
	}
	tree, err := NewTree(txs)
	if err != nil {
		t.Fatal(err)
	}
	isTxInBlock, err := tree.VerifyContent(TestSHA256Content{x: test_tx})
	if err != nil {
		t.Fatal(err)
	}
	if !isTxInBlock {
		t.Fatal(err)
	}
}
