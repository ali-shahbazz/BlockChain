package crypto

import (
	"bytes"
	"crypto/sha256"
)

// MerkleNode represents a node in a Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	RootNode *MerkleNode
}

// NewMerkleNode creates a new Merkle tree node
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	if left == nil && right == nil {
		// Leaf node, hash the data
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		// Internal node, combine left and right children hashes
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return &node
}

// NewMerkleTree creates a new Merkle tree from a sequence of data
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode

	// Create leaf nodes
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, node)
	}

	// Handle odd number of nodes by duplicating the last one
	if len(nodes)%2 != 0 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}

	// Build tree bottom-up
	for len(nodes) > 1 {
		var level []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			node := NewMerkleNode(nodes[i], nodes[i+1], nil)
			level = append(level, node)
		}

		// Ensure even number of nodes at each level
		if len(level)%2 != 0 && len(level) > 1 {
			level = append(level, level[len(level)-1])
		}

		nodes = level
	}

	return &MerkleTree{RootNode: nodes[0]}
}

// GetRootHash returns the root hash of the Merkle tree
func (m *MerkleTree) GetRootHash() []byte {
	return m.RootNode.Data
}

// VerifyData verifies if some data is in the tree
func (m *MerkleTree) VerifyData(data []byte) bool {
	hash := sha256.Sum256(data)
	return m.verifyHash(hash[:], m.RootNode)
}

// verifyHash is a helper function to verify a hash in the tree
func (m *MerkleTree) verifyHash(hash []byte, node *MerkleNode) bool {
	if node == nil {
		return false
	}

	if bytes.Equal(node.Data, hash) {
		return true
	}

	if node.Left == nil && node.Right == nil {
		return false
	}

	return m.verifyHash(hash, node.Left) || m.verifyHash(hash, node.Right)
}

// GenerateProof generates a Merkle proof for the given data
func (m *MerkleTree) GenerateProof(data []byte) [][]byte {
	hash := sha256.Sum256(data)
	proof := [][]byte{}
	m.generateProofHelper(hash[:], m.RootNode, &proof, []byte{})
	return proof
}

// generateProofHelper is a recursive helper for generating proofs
func (m *MerkleTree) generateProofHelper(hash []byte, node *MerkleNode, proof *[][]byte, side []byte) bool {
	if node == nil {
		return false
	}

	// If we're at a leaf node and it's our target, we found the path
	if node.Left == nil && node.Right == nil {
		if bytes.Equal(node.Data, hash) {
			return true
		}
		return false
	}

	// Try left subtree
	if m.generateProofHelper(hash, node.Left, proof, []byte{0}) {
		// Add the right node to the proof
		*proof = append(*proof, node.Right.Data)
		*proof = append(*proof, []byte{1}) // 1 indicates right side
		return true
	}

	// Try right subtree
	if m.generateProofHelper(hash, node.Right, proof, []byte{1}) {
		// Add the left node to the proof
		*proof = append(*proof, node.Left.Data)
		*proof = append(*proof, []byte{0}) // 0 indicates left side
		return true
	}

	return false
}

// VerifyProof verifies a Merkle proof
func VerifyProof(data []byte, proof [][]byte, rootHash []byte) bool {
	hash := sha256.Sum256(data)
	currentHash := hash[:]

	for i := 0; i < len(proof); i += 2 {
		side := proof[i+1][0]
		proofElement := proof[i]

		if side == 0 {
			// The proof element is the left sibling
			combined := append(proofElement, currentHash...)
			h := sha256.Sum256(combined)
			currentHash = h[:]
		} else {
			// The proof element is the right sibling
			combined := append(currentHash, proofElement...)
			h := sha256.Sum256(combined)
			currentHash = h[:]
		}
	}

	return bytes.Equal(currentHash, rootHash)
}
