package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"sync"
)

// Blockchain represents a chain of blocks
type Blockchain struct {
	blocks         []*Block
	currentHeight  int64
	latestBlockMap map[string]*Block // Maps block hashes to blocks
	mutex          sync.RWMutex
}

// NewBlockchain creates a new blockchain with genesis block
func NewBlockchain() *Blockchain {
	genesisData := []byte("Genesis Block for Advanced Blockchain System")
	genesisTx := NewTransaction(genesisData)
	genesisBlock := NewGenesisBlock(genesisTx)

	bc := &Blockchain{
		blocks:         []*Block{genesisBlock},
		currentHeight:  0,
		latestBlockMap: make(map[string]*Block),
	}
	bc.latestBlockMap[hex.EncodeToString(genesisBlock.Hash)] = genesisBlock

	return bc
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(transactions []*Transaction) (*Block, error) {
	// Simple synchronous implementation to avoid goroutine/lock issues
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	log.Printf("Adding new block to blockchain at height %d", bc.currentHeight+1)

	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(transactions, prevBlock.Hash, prevBlock.Height+1)

	// Basic validation before adding
	if !bc.validateBlock(newBlock) {
		return nil, errors.New("invalid block")
	}

	// Add the block to the chain
	bc.blocks = append(bc.blocks, newBlock)
	bc.currentHeight = newBlock.Height
	bc.latestBlockMap[hex.EncodeToString(newBlock.Hash)] = newBlock

	log.Printf("Successfully added block %d with hash %x", newBlock.Height, newBlock.Hash)

	return newBlock, nil
}

// GetLatestBlock returns the latest block in the chain
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

// GetBlockByHash returns a block by its hash
func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	blockHash := hex.EncodeToString(hash)
	if block, exists := bc.latestBlockMap[blockHash]; exists {
		return block
	}
	return nil
}

// GetBlockByHeight returns a block by its height
func (bc *Blockchain) GetBlockByHeight(height int64) *Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	if height < 0 || height > bc.currentHeight {
		return nil
	}

	return bc.blocks[height]
}

// GetHeight returns the current height of the blockchain
func (bc *Blockchain) GetHeight() int64 {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return bc.currentHeight
}

// validateBlock performs basic validation on a block
func (bc *Blockchain) validateBlock(block *Block) bool {
	// Check if the block's previous hash matches the hash of the latest block
	prevBlock := bc.GetLatestBlock()
	if prevBlock == nil {
		// Only valid for genesis block
		return block.Height == 0
	}

	if !bytes.Equal(block.PrevBlockHash, prevBlock.Hash) {
		return false
	}

	if block.Height != prevBlock.Height+1 {
		return false
	}

	// In a real implementation, you'd validate the block's hash
	// against the difficulty target, check transactions, etc.

	return true
}

// Iterator returns a BlockchainIterator to iterate over blocks
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return &BlockchainIterator{
		currentHash: bc.GetLatestBlock().Hash,
		blockchain:  bc,
	}
}

// BlockchainIterator is used to iterate over blockchain blocks
type BlockchainIterator struct {
	currentHash []byte
	blockchain  *Blockchain
}

// Next returns the next block in the blockchain
func (i *BlockchainIterator) Next() *Block {
	block := i.blockchain.GetBlockByHash(i.currentHash)

	if block == nil {
		return nil
	}

	i.currentHash = block.PrevBlockHash
	return block
}
