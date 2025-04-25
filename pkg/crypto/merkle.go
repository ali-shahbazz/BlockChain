package crypto

import (
	"bytes"
	"crypto/sha256"
	"sync"
)

// MerkleNode represents a node in a Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	RootNode  *MerkleNode
	leafNodes []*MerkleNode
	hashCache map[string][]byte
	cacheMux  sync.RWMutex
}

// NewMerkleTree creates a new Merkle tree from a sequence of data
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode
	var wg sync.WaitGroup
	nodesChan := make(chan *MerkleNode, len(data))

	// Create leaf nodes in parallel for better performance with large datasets
	for _, datum := range data {
		wg.Add(1)
		go func(datum []byte) {
			defer wg.Done()
			nodesChan <- &MerkleNode{
				Left:  nil,
				Right: nil,
				Data:  calculateHash(datum),
			}
		}(datum)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(nodesChan)
	}()

	// Collect nodes from channel
	for node := range nodesChan {
		nodes = append(nodes, node)
	}

	// If there are no nodes, return a tree with a nil root
	if len(nodes) == 0 {
		return &MerkleTree{
			RootNode:  nil,
			leafNodes: []*MerkleNode{},
			hashCache: make(map[string][]byte),
		}
	}

	// Use parallel processing for building tree levels when there are many nodes
	if len(nodes) > 1000 {
		return buildTreeInParallel(nodes)
	} else {
		return buildTreeSequentially(nodes)
	}
}

// buildTreeSequentially builds a Merkle tree sequentially (for smaller trees)
func buildTreeSequentially(nodes []*MerkleNode) *MerkleTree {
	// Store leaf nodes for verification
	leafNodes := make([]*MerkleNode, len(nodes))
	copy(leafNodes, nodes)

	// Build the tree level by level from the bottom up
	for len(nodes) > 1 {
		var newLevel []*MerkleNode

		// Handle odd number of nodes by duplicating the last one
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		// Create parent nodes for each pair
		for i := 0; i < len(nodes); i += 2 {
			node := &MerkleNode{
				Left:  nodes[i],
				Right: nodes[i+1],
				Data:  calculateHash(append(nodes[i].Data, nodes[i+1].Data...)),
			}
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	// The root is the only node left
	return &MerkleTree{
		RootNode:  nodes[0],
		leafNodes: leafNodes,
		hashCache: make(map[string][]byte),
	}
}

// buildTreeInParallel builds a Merkle tree using parallel processing (for larger trees)
func buildTreeInParallel(nodes []*MerkleNode) *MerkleTree {
	// Store leaf nodes for verification
	leafNodes := make([]*MerkleNode, len(nodes))
	copy(leafNodes, nodes)

	// Build the tree level by level from the bottom up
	for len(nodes) > 1 {
		// Handle odd number of nodes by duplicating the last one
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		var wg sync.WaitGroup
		newLevel := make([]*MerkleNode, len(nodes)/2)

		// Create parent nodes for each pair in parallel
		for i := 0; i < len(nodes); i += 2 {
			wg.Add(1)
			go func(i int, idx int) {
				defer wg.Done()
				newLevel[idx] = &MerkleNode{
					Left:  nodes[i],
					Right: nodes[i+1],
					Data:  calculateHash(append(nodes[i].Data, nodes[i+1].Data...)),
				}
			}(i, i/2)
		}

		wg.Wait()
		nodes = newLevel
	}

	// The root is the only node left
	return &MerkleTree{
		RootNode:  nodes[0],
		leafNodes: leafNodes,
		hashCache: make(map[string][]byte),
	}
}

// GetRootHash returns the root hash of the Merkle tree
func (m *MerkleTree) GetRootHash() []byte {
	if m.RootNode == nil {
		return nil
	}
	return m.RootNode.Data
}

// VerifyData checks if data is in the Merkle tree
func (m *MerkleTree) VerifyData(data []byte) bool {
	dataHash := calculateHash(data)

	// Use cached result if available
	cacheKey := string(dataHash)
	m.cacheMux.RLock()
	if cachedResult, exists := m.hashCache[cacheKey]; exists {
		m.cacheMux.RUnlock()
		return bytes.Equal(cachedResult, m.GetRootHash())
	}
	m.cacheMux.RUnlock()

	// Check each leaf node
	for _, leaf := range m.leafNodes {
		if bytes.Equal(leaf.Data, dataHash) {
			// Cache the result for future verifications
			m.cacheMux.Lock()
			m.hashCache[cacheKey] = m.GetRootHash()
			m.cacheMux.Unlock()
			return true
		}
	}

	return false
}

// GenerateProof generates a Merkle proof for the given data
func (m *MerkleTree) GenerateProof(data []byte) [][]byte {
	if m.RootNode == nil {
		return nil
	}

	dataHash := calculateHash(data)
	var proof [][]byte
	var path []bool // left or right for each level

	// Find the leaf node containing this data
	var leafIndex int = -1
	for i, leaf := range m.leafNodes {
		if bytes.Equal(leaf.Data, dataHash) {
			leafIndex = i
			break
		}
	}

	if leafIndex == -1 {
		return nil // Data not found
	}

	// Compute path from leaf to root
	nodes := m.leafNodes
	currentIndex := leafIndex

	for len(nodes) > 1 {
		// If odd number of nodes, duplicate the last one
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
			// If we're at the last position, update currentIndex
			if currentIndex == len(nodes)-2 {
				currentIndex = len(nodes) - 1
			}
		}

		isOdd := currentIndex%2 != 0
		siblingIndex := currentIndex
		if isOdd {
			siblingIndex--
		} else {
			siblingIndex++
		}

		// Add sibling to proof
		proof = append(proof, nodes[siblingIndex].Data)
		// Add direction to path
		path = append(path, isOdd)

		// Move to the parent node
		var newNodes []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			newNode := &MerkleNode{
				Left:  nodes[i],
				Right: nodes[i+1],
				Data:  calculateHash(append(nodes[i].Data, nodes[i+1].Data...)),
			}
			newNodes = append(newNodes, newNode)
		}

		nodes = newNodes
		currentIndex = currentIndex / 2
	}

	// Encode path into proof
	pathBytes := make([]byte, (len(path)+7)/8) // Bit-packed representation
	for i, isRight := range path {
		if isRight {
			byteIndex := i / 8
			bitPosition := i % 8
			pathBytes[byteIndex] |= (1 << bitPosition)
		}
	}

	// Prepend path to proof
	return append([][]byte{pathBytes}, proof...)
}

// VerifyProof verifies a Merkle proof against a root hash
func VerifyProof(data []byte, proof [][]byte, rootHash []byte) bool {
	if len(proof) == 0 {
		return false
	}

	// Extract path from proof
	pathBytes := proof[0]
	siblings := proof[1:]

	// Calculate hash of data
	currentHash := calculateHash(data)

	// Replay the proof
	for i, sibling := range siblings {
		isRight := false
		if i/8 < len(pathBytes) {
			isRight = (pathBytes[i/8] & (1 << (i % 8))) != 0
		}

		if isRight {
			// Data is on the right side
			currentHash = calculateHash(append(sibling, currentHash...))
		} else {
			// Data is on the left side
			currentHash = calculateHash(append(currentHash, sibling...))
		}
	}

	// Final hash should match root hash
	return bytes.Equal(currentHash, rootHash)
}

// calculateHash calculates the SHA-256 hash of data
func calculateHash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
