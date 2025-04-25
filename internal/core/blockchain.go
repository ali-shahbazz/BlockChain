package core

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
)

// Blockchain represents the main blockchain structure
type Blockchain struct {
	mutex       sync.RWMutex
	persistence *BlockchainPersistence
	tip         []byte // hash of the latest block
}

// NewBlockchain creates a new blockchain with a genesis block
func NewBlockchain() *Blockchain {
	return NewBlockchainWithPath("./data/blockchain")
}

// NewBlockchainWithPath creates a new blockchain with a custom data directory
func NewBlockchainWithPath(dataDir string) *Blockchain {
	// Create persistence manager with the specified data directory
	persistence, err := NewBlockchainPersistence(dataDir)
	if err != nil {
		log.Printf("Error creating blockchain persistence: %v", err)
		// Fall back to in-memory mode
		return newInMemoryBlockchain()
	}

	// Check if we have an existing chain
	latestBlock, err := persistence.GetLatestBlock()
	if err != nil {
		// No existing chain, create a genesis block
		coinbase := NewTransaction([]byte("Genesis Block"))
		genesis := NewGenesisBlock(coinbase)

		err = persistence.SaveBlock(genesis)
		if err != nil {
			log.Printf("Error saving genesis block: %v", err)
			return newInMemoryBlockchain()
		}

		return &Blockchain{
			persistence: persistence,
			tip:         genesis.Hash,
		}
	}

	// Existing chain found
	return &Blockchain{
		persistence: persistence,
		tip:         latestBlock.Hash,
	}
}

// newInMemoryBlockchain creates a new in-memory blockchain
func newInMemoryBlockchain() *Blockchain {
	coinbase := NewTransaction([]byte("Genesis Block"))
	genesis := NewGenesisBlock(coinbase)

	// Create a memory-only persistence
	memPersistence := &BlockchainPersistence{
		blocksInMemCache: make(map[string]*Block),
		heightCache:      make(map[int64]*Block),
		fileMap:          make(map[int]*BlockFile),
	}

	// Add genesis block to in-memory cache
	memPersistence.blocksInMemCache[string(genesis.Hash)] = genesis
	memPersistence.heightCache[genesis.Height] = genesis

	return &Blockchain{
		persistence: memPersistence,
		tip:         genesis.Hash,
	}
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(transactions []*Transaction) (*Block, error) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	lastHash := bc.tip
	lastBlock, err := bc.persistence.GetBlockByHash(lastHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get last block: %w", err)
	}

	newBlock := NewBlock(transactions, lastHash, lastBlock.Height+1)

	// Add proof-of-work (simple implementation)
	err = bc.mineBlock(newBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to mine block: %w", err)
	}

	// Save the block to storage
	err = bc.persistence.SaveBlock(newBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to save block: %w", err)
	}

	// Update the blockchain tip
	bc.tip = newBlock.Hash

	return newBlock, nil
}

// mineBlock performs proof-of-work on a block
func (bc *Blockchain) mineBlock(block *Block) error {
	targetBits := 16 // Reduced difficulty for testing (was 24)
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	var hashInt big.Int
	nonce := int64(0)
	maxNonce := int64(1000000) // Limit to prevent infinite loop

	log.Printf("Mining block with %d transactions", len(block.Transactions))

	for nonce < maxNonce {
		block.Nonce = nonce

		// Recalculate the block hash with the new nonce
		block.SetHash()

		// Convert block hash to big.Int for comparison
		hashInt.SetBytes(block.Hash)

		// Check if hash is less than target (meets difficulty)
		if hashInt.Cmp(target) == -1 {
			log.Printf("Block mined with nonce %d: %x", nonce, block.Hash)
			return nil
		}

		nonce++
	}

	return errors.New("failed to mine block: exceeded maximum nonce")
}

// GetBlockByHash returns a block by its hash
func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	block, err := bc.persistence.GetBlockByHash(hash)
	if err != nil {
		return nil
	}
	return block
}

// GetBlockByHeight returns a block by its height
func (bc *Blockchain) GetBlockByHeight(height int64) *Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	block, err := bc.persistence.GetBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}

// GetHeight returns the current height of the blockchain
func (bc *Blockchain) GetHeight() int64 {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	height, err := bc.persistence.GetBlockHeight()
	if err != nil {
		return 0 // Default to genesis block height if there's an error
	}
	return height
}

// Iterator creates a blockchain iterator
func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{
		currentHash: bc.tip,
		bc:          bc,
	}
}

// Close properly closes the blockchain resources
func (bc *Blockchain) Close() error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	return bc.persistence.Close()
}

// BlockchainIterator is used to iterate over blockchain blocks
type BlockchainIterator struct {
	currentHash []byte
	bc          *Blockchain
}

// Next returns the next block starting from the tip
func (i *BlockchainIterator) Next() *Block {
	if len(i.currentHash) == 0 {
		return nil
	}

	block := i.bc.GetBlockByHash(i.currentHash)
	if block == nil {
		return nil
	}

	i.currentHash = block.PrevBlockHash
	return block
}

// ValidateChain checks if the blockchain is valid
func (bc *Blockchain) ValidateChain() (bool, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	height := bc.GetHeight()

	// Get blocks in batches to validate
	batchSize := int64(100)
	for start := int64(0); start <= height; start += batchSize {
		end := start + batchSize
		if end > height {
			end = height
		}

		blocks, err := bc.persistence.GetBlocksBetweenHeights(start, end)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve blocks: %w", err)
		}

		// Validate each block
		for i, block := range blocks {
			// Skip genesis block
			if block.Height == 0 {
				continue
			}

			// Get previous block
			var prevBlock *Block
			if i > 0 && blocks[i-1].Height == block.Height-1 {
				// Previous block is in our batch
				prevBlock = blocks[i-1]
			} else {
				// Need to fetch previous block
				prevBlock, err = bc.persistence.GetBlockByHeight(block.Height - 1)
				if err != nil {
					return false, fmt.Errorf("failed to get previous block: %w", err)
				}
			}

			// Validate block references previous block's hash
			if !bytes.Equal(block.PrevBlockHash, prevBlock.Hash) {
				return false, fmt.Errorf("invalid previous block hash at height %d", block.Height)
			}

			// Validate block hash is correct
			calculatedHash := block.CalculateHash()
			if !bytes.Equal(calculatedHash, block.Hash) {
				return false, fmt.Errorf("invalid block hash at height %d", block.Height)
			}
		}
	}

	return true, nil
}

// GetRecentBlocks returns the most recent N blocks
func (bc *Blockchain) GetRecentBlocks(count int) ([]*Block, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	height := bc.GetHeight()
	startHeight := height - int64(count) + 1
	if startHeight < 0 {
		startHeight = 0
	}

	return bc.persistence.GetBlocksBetweenHeights(startHeight, height)
}
