package crypto

import (
	"testing"
)

// TestNewMerkleTree verifies Merkle tree creation
func TestNewMerkleTree(t *testing.T) {
	data := [][]byte{
		[]byte("Transaction 1"),
		[]byte("Transaction 2"),
		[]byte("Transaction 3"),
	}

	tree := NewMerkleTree(data)

	if tree.RootNode == nil {
		t.Errorf("Expected Merkle tree to have a root node")
	}

	if len(tree.GetRootHash()) == 0 {
		t.Errorf("Expected Merkle tree to have a non-empty root hash")
	}
}

// TestVerifyData verifies data verification in a Merkle tree
func TestVerifyData(t *testing.T) {
	tx1 := []byte("Transaction 1")
	tx2 := []byte("Transaction 2")
	tx3 := []byte("Transaction 3")

	data := [][]byte{tx1, tx2, tx3}
	tree := NewMerkleTree(data)

	// Data in the tree should be verified
	if !tree.VerifyData(tx1) {
		t.Errorf("Failed to verify tx1 which should be in the tree")
	}

	if !tree.VerifyData(tx2) {
		t.Errorf("Failed to verify tx2 which should be in the tree")
	}

	if !tree.VerifyData(tx3) {
		t.Errorf("Failed to verify tx3 which should be in the tree")
	}

	// Data not in the tree should not be verified
	tx4 := []byte("Transaction 4")
	if tree.VerifyData(tx4) {
		t.Errorf("Incorrectly verified tx4 which should not be in the tree")
	}
}

// TestGenerateAndVerifyProof tests proof generation and verification
func TestGenerateAndVerifyProof(t *testing.T) {
	tx1 := []byte("Transaction 1")
	tx2 := []byte("Transaction 2")
	tx3 := []byte("Transaction 3")
	tx4 := []byte("Transaction 4")

	data := [][]byte{tx1, tx2, tx3}
	tree := NewMerkleTree(data)

	// Generate proof for tx2
	proof := tree.GenerateProof(tx2)

	// Verify the proof
	if !VerifyProof(tx2, proof, tree.GetRootHash()) {
		t.Errorf("Failed to verify a valid Merkle proof")
	}

	// Verify that the proof fails for the wrong data
	if VerifyProof(tx4, proof, tree.GetRootHash()) {
		t.Errorf("Incorrectly verified a proof with wrong data")
	}

	// Verify that the proof fails for the wrong root hash
	wrongRootHash := []byte("wrong hash")
	if VerifyProof(tx2, proof, wrongRootHash) {
		t.Errorf("Incorrectly verified a proof with wrong root hash")
	}
}
