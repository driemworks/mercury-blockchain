package state

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"hash"
	"log"
)

// credit goes to: https://github.com/web3coach/merkletree/blob/master/merkle_tree.go
type Content interface {
	CalculateHash() ([]byte, error)
	Equals(other Content) (bool, error)
}

type MerkleTree struct {
	Root         *TreeNode
	merkleRoot   []byte
	Leaves       []*TreeNode
	hashStrategy func() hash.Hash
}

type TreeNode struct {
	Tree    *MerkleTree
	Parent  *TreeNode
	Left    *TreeNode
	Right   *TreeNode
	leaf    bool
	dup     bool
	Hash    []byte
	content Content
}

func NewTree(contents []Content) (*MerkleTree, error) {
	var defaultHashStrategy = sha256.New
	tree := &MerkleTree{
		hashStrategy: defaultHashStrategy,
	}
	root, leaves, err := buildWithContents(contents, tree)
	if err != nil {
		return nil, err
	}
	tree.Root = root
	tree.Leaves = leaves
	tree.merkleRoot = root.Hash
	return tree, nil
}

/*
	This function is used for Merkle Proofs
	1) Find leaf node storing the tx we want to verify
	2) Hash the identified leaf along with sibling leaves to generate the parent node
	3) find the parent's sibling node
	4) repeat 2 and 3 until you reach the root
	5) if the generated merkle root matched the block's expected merkle root then the tx is in the block
*/
func (t *MerkleTree) VerifyContent(content Content) (bool, error) {
	for _, l := range t.Leaves {
		// find leaf node
		ok, err := l.content.Equals(content)
		if err != nil {
			return false, err
		}
		if ok {
			currentParent := l.Parent
			for currentParent != nil {
				h := t.hashStrategy()
				rightBytes, err := currentParent.Right.calculateNodeHash()
				if err != nil {
					return false, err
				}
				leftBytes, err := currentParent.Left.calculateNodeHash()
				if err != nil {
					return false, err
				}

				if _, err := h.Write(append(leftBytes, rightBytes...)); err != nil {
					return false, err
				}
				if bytes.Compare(h.Sum(nil), currentParent.Hash) != 0 {
					return false, nil
				}
				currentParent = currentParent.Parent
			}
			return true, nil
		}
	}
	return false, nil
}

func (n *TreeNode) calculateNodeHash() ([]byte, error) {
	if n.leaf {
		return n.content.CalculateHash()
	}
	h := n.Tree.hashStrategy()
	if _, err := h.Write(append(n.Left.Hash, n.Right.Hash...)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func buildWithContents(cs []Content, t *MerkleTree) (*TreeNode, []*TreeNode, error) {
	if len(cs) == 0 {
		return nil, nil, errors.New("cannot construct tree with no content")
	}
	var leaves []*TreeNode
	for _, c := range cs {
		hash, err := c.CalculateHash()
		if err != nil {
			return nil, nil, err
		}
		leaves = append(leaves, &TreeNode{
			Hash:    hash,
			content: c,
			leaf:    true,
			Tree:    t,
		})
	}
	if len(leaves)%2 == 1 {
		// create a duplicate to pair with the already created node
		duplicate := &TreeNode{
			Hash:    leaves[len(leaves)-1].Hash,
			content: leaves[len(leaves)-1].content,
			leaf:    true,
			dup:     true,
			Tree:    t,
		}
		leaves = append(leaves, duplicate)
	}
	root, err := buildIntermediate(leaves, t)
	if err != nil {
		return nil, nil, err
	}
	return root, leaves, nil
}

func buildIntermediate(nodeList []*TreeNode, t *MerkleTree) (*TreeNode, error) {
	var nodes []*TreeNode
	for i := 0; i < len(nodeList); i += 2 {
		h := t.hashStrategy()
		var left, right int = i, i + 1
		// in a merkle tree, if there is no node with which to pair a node, pair it with a copy of itself
		if i+1 == len(nodeList) {
			right = i
		}
		c_hash := append(nodeList[left].Hash, nodeList[right].Hash...)
		if _, err := h.Write(c_hash); err != nil {
			log.Fatalln(err)
		}
		tn := &TreeNode{
			Left:  nodeList[left],
			Right: nodeList[right],
			Hash:  h.Sum(nil),
			Tree:  t,
		}
		nodes = append(nodes, tn)
		nodeList[left].Parent = tn
		nodeList[right].Parent = tn
		if len(nodeList) == 2 {
			return tn, nil
		}
	}

	return buildIntermediate(nodes, t)
}
