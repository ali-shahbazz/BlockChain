package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"time"

	"blockchain-a3/pkg/crypto"
)

// Block represents a single block in the blockchain
type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Height        int64
	Nonce         int64

	// Added fields for advanced blockchain features
	MerkleRoot       []byte            // Root hash of the Merkle tree
	StateRoot        []byte            // Root hash of the state trie
	AccumulatorRoot  []byte            // Cryptographic accumulator root
	EntropyFactor    float64           // Measure of randomness (0.0-1.0)
	VectorClock      map[string]uint64 // For causal consistency
	ValidationProofs [][]byte          // Proofs for state validation
	ConsensusData    []byte            // Consensus-specific metadata
	ShardID          string            // Shard identifier
	CrossShardRefs   []string          // References to cross-shard transactions
	Signature        []byte            // Block proposer signature
	ValidatorSigs    map[string][]byte // Signatures from validators
}

// NewBlock creates a new block instance with enhanced features
func NewBlock(transactions []*Transaction, prevBlockHash []byte, height int64) *Block {
	block := &Block{
		Timestamp:        time.Now().UnixNano(),
		Transactions:     transactions,
		PrevBlockHash:    prevBlockHash,
		Height:           height,
		Nonce:            0,
		VectorClock:      make(map[string]uint64),
		ValidationProofs: make([][]byte, 0),
		ValidatorSigs:    make(map[string][]byte),
	}

	// Generate simplified Merkle root synchronously to avoid goroutine issues
	var txHashes [][]byte
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.ID)
	}

	if len(txHashes) > 0 {
		merkleTree := crypto.NewMerkleTree(txHashes)
		block.MerkleRoot = merkleTree.GetRootHash()
	}

	// Calculate initial hash
	block.SetHash()

	// Calculate entropy factor
	block.EntropyFactor = calculateEntropyFactor(block.Hash)

	return block
}

// NewGenesisBlock creates the genesis (first) block in the blockchain
func NewGenesisBlock(coinbase *Transaction) *Block {
	// Create genesis block with special settings
	genesis := &Block{
		Timestamp:        time.Now().UnixNano(),
		Transactions:     []*Transaction{coinbase},
		PrevBlockHash:    []byte{}, // Empty for genesis block
		Height:           0,        // Genesis is always height 0
		Nonce:            0,
		VectorClock:      make(map[string]uint64),
		ValidationProofs: make([][]byte, 0),
		ValidatorSigs:    make(map[string][]byte),
	}

	// Generate Merkle root from transactions
	genesis.MerkleRoot = genesis.generateMerkleRoot()

	// Calculate initial hash
	genesis.SetHash()

	// Calculate entropy factor
	genesis.EntropyFactor = calculateEntropyFactor(genesis.Hash)

	return genesis
}

// generateMerkleRoot creates a proper Merkle tree and returns its root
func (b *Block) generateMerkleRoot() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	if len(txHashes) == 0 {
		return nil
	}

	// Use the Merkle tree implementation from crypto package
	merkleTree := crypto.NewMerkleTree(txHashes)
	return merkleTree.GetRootHash()
}

// SetHash calculates the hash of the block with advanced fields
func (b *Block) SetHash() {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	// Encode block headers including new fields
	err := enc.Encode(struct {
		Timestamp       int64
		PrevBlockHash   []byte
		MerkleRoot      []byte
		Height          int64
		Nonce           int64
		StateRoot       []byte
		AccumulatorRoot []byte
		ShardID         string
	}{ // Fixed syntax error: added proper opening brace here
		Timestamp:       b.Timestamp,
		PrevBlockHash:   b.PrevBlockHash,
		MerkleRoot:      b.MerkleRoot,
		Height:          b.Height,
		Nonce:           b.Nonce,
		StateRoot:       b.StateRoot,
		AccumulatorRoot: b.AccumulatorRoot,
		ShardID:         b.ShardID,
	})

	if err != nil {
		log.Panicf("Failed to encode block header: %v", err)
	}

	hash := sha256.Sum256(encoded.Bytes())
	b.Hash = hash[:]
}

// SignBlock signs the block with the given private key
func (b *Block) SignBlock(privateKey []byte) error {
	// In a real implementation, this would use proper digital signatures
	// For now, we'll use a simplified approach

	signatureInput := append(b.Hash, privateKey...)
	signature := sha256.Sum256(signatureInput)
	b.Signature = signature[:]

	return nil
}

// VerifyBlockSignature verifies the block's signature
func (b *Block) VerifyBlockSignature(publicKey []byte) bool {
	// In a real implementation, this would verify against the public key
	// For now, we'll use a simplified approach for demonstration

	signatureInput := append(b.Hash, publicKey...)
	expectedSignature := sha256.Sum256(signatureInput)

	return bytes.Equal(b.Signature, expectedSignature[:])
}

// AddValidatorSignature adds a validator's signature to the block
func (b *Block) AddValidatorSignature(validatorID string, signature []byte) {
	// Ensure map is initialized
	if b.ValidatorSigs == nil {
		b.ValidatorSigs = make(map[string][]byte)
	}
	b.ValidatorSigs[validatorID] = signature
}

// CalculateAccumulatorRoot computes a cryptographic accumulator for the block data
func (b *Block) CalculateAccumulatorRoot() error {
	// Create elements for the accumulator from block data
	elements := make(map[string][]byte)

	// Add transaction data
	for i, tx := range b.Transactions {
		key := fmt.Sprintf("tx-%d", i)
		elements[key] = tx.ID
	}

	// Add block header elements
	elements["timestamp"] = int64ToBytes(b.Timestamp)
	elements["height"] = int64ToBytes(b.Height)
	elements["merkle_root"] = b.MerkleRoot

	// Create a new accumulator
	acc, err := crypto.NewCryptographicAccumulator(2048)
	if err != nil {
		return err
	}

	// Add all elements to the accumulator
	for _, data := range elements {
		acc.Add(data)
	}

	// Set the accumulator root using the GetValue() method
	b.AccumulatorRoot = acc.GetValue().Bytes()

	return nil
}

// GenerateStateProof generates a proof for the block's state
func (b *Block) GenerateStateProof() ([]byte, error) {
	// In a real implementation, this would generate a compact proof of the block's state
	// For now, we'll create a simple hash-based proof

	stateData := append(b.Hash, b.MerkleRoot...)
	stateData = append(stateData, b.AccumulatorRoot...)

	proof := sha256.Sum256(stateData)
	return proof[:], nil
}

// VerifyProof verifies a provided proof against the block's state
func (b *Block) VerifyProof(proof []byte, dataHash []byte) bool {
	// In a real implementation, this would verify the proof against the block's state
	// For now, we'll provide a simple verification method

	// Reconstruct expected proof
	stateData := append(b.Hash, dataHash...)
	expectedProof := sha256.Sum256(stateData)

	return bytes.Equal(proof, expectedProof[:])
}

// GetEntropyScore returns a score based on the entropy of the block
func (b *Block) GetEntropyScore() float64 {
	// Return the pre-calculated entropy factor
	return b.EntropyFactor
}

// calculateEntropyFactor derives a normalized entropy factor from a hash
func calculateEntropyFactor(hash []byte) float64 {
	// Calculate Shannon entropy
	counts := make(map[byte]int)
	for _, b := range hash {
		counts[b]++
	}

	entropy := 0.0
	for _, count := range counts {
		prob := float64(count) / float64(len(hash))
		entropy -= prob * math.Log2(prob)
	}

	// Normalize to [0, 1]
	maxEntropy := math.Log2(256.0) // Max possible entropy for bytes
	return entropy / maxEntropy
}

// int64ToBytes converts an int64 to bytes
func int64ToBytes(val int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return buf
}
