package core

import (
	"os"
	"testing"
)

// TestBlockchainPersistence tests the blockchain persistence functionalities
func TestBlockchainPersistence(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "blockchain-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a new persistence manager
	persistence, err := NewBlockchainPersistence(tempDir)
	if err != nil {
		t.Fatalf("Failed to create persistence manager: %v", err)
	}

	// Test saving and retrieving blocks
	coinbase := NewTransaction([]byte("Test Genesis Block"))
	genesis := NewGenesisBlock(coinbase)

	// Save genesis block
	if err := persistence.SaveBlock(genesis); err != nil {
		t.Fatalf("Failed to save genesis block: %v", err)
	}

	// Create and save additional blocks
	blocks := []*Block{genesis}
	for i := 1; i < 5; i++ {
		tx := NewTransaction([]byte("Test Block Data"))
		prevHash := blocks[i-1].Hash
		block := NewBlock([]*Transaction{tx}, prevHash, int64(i))
		blocks = append(blocks, block)

		// Save the block
		if err := persistence.SaveBlock(block); err != nil {
			t.Fatalf("Failed to save block %d: %v", i, err)
		}
	}

	// Test retrieving by hash
	for _, block := range blocks {
		retrieved, err := persistence.GetBlockByHash(block.Hash)
		if err != nil {
			t.Errorf("Failed to retrieve block by hash: %v", err)
			continue
		}

		if string(retrieved.Hash) != string(block.Hash) {
			t.Errorf("Retrieved block hash doesn't match: expected %x, got %x", block.Hash, retrieved.Hash)
		}
	}

	// Test retrieving by height
	for i, block := range blocks {
		retrieved, err := persistence.GetBlockByHeight(int64(i))
		if err != nil {
			t.Errorf("Failed to retrieve block by height %d: %v", i, err)
			continue
		}

		if string(retrieved.Hash) != string(block.Hash) {
			t.Errorf("Retrieved block hash doesn't match for height %d: expected %x, got %x", i, block.Hash, retrieved.Hash)
		}
	}

	// Test getting latest block
	latest, err := persistence.GetLatestBlock()
	if err != nil {
		t.Errorf("Failed to get latest block: %v", err)
	} else if latest.Height != 4 {
		t.Errorf("Latest block has incorrect height: expected 4, got %d", latest.Height)
	}

	// Test GetBlocksBetweenHeights
	rangeBlocks, err := persistence.GetBlocksBetweenHeights(1, 3)
	if err != nil {
		t.Errorf("Failed to get blocks between heights: %v", err)
	} else if len(rangeBlocks) != 3 {
		t.Errorf("Incorrect number of blocks returned: expected 3, got %d", len(rangeBlocks))
	}

	// Test Close and reopening
	err = persistence.Close()
	if err != nil {
		t.Errorf("Failed to close persistence: %v", err)
	}

	// Reopen the persistence manager and check if data is still there
	persistence2, err := NewBlockchainPersistence(tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen persistence manager: %v", err)
	}

	// Check if blocks can still be retrieved
	for i := 0; i < 5; i++ {
		retrieved, err := persistence2.GetBlockByHeight(int64(i))
		if err != nil {
			t.Errorf("Failed to retrieve block by height %d after reopening: %v", i, err)
		} else if retrieved.Height != int64(i) {
			t.Errorf("Block has incorrect height after reopening: expected %d, got %d", i, retrieved.Height)
		}
	}
}

// TestPersistentBlockchain tests the blockchain with persistence
func TestPersistentBlockchain(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "blockchain-full-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a blockchain with our temporary directory
	bc := NewBlockchainWithPath(tempDir)

	// Add a few blocks
	for i := 0; i < 3; i++ {
		tx := NewTransaction([]byte("Persistent Block Data"))
		_, err := bc.AddBlock([]*Transaction{tx})
		if err != nil {
			t.Errorf("Failed to add block %d: %v", i+1, err)
		}
	}

	// Check height
	if bc.GetHeight() != 3 {
		t.Errorf("Expected blockchain height to be 3, got %d", bc.GetHeight())
	}

	// Close the blockchain to ensure data is persisted
	bc.Close()

	// Create a new blockchain instance which should load from disk
	bc2 := NewBlockchainWithPath(tempDir)

	// Verify the height is maintained
	if bc2.GetHeight() != 3 {
		t.Errorf("Expected blockchain height to be 3 after reopening, got %d", bc2.GetHeight())
	}

	// Verify we can retrieve blocks
	for i := int64(0); i <= 3; i++ {
		block := bc2.GetBlockByHeight(i)
		if block == nil {
			t.Errorf("Failed to retrieve block at height %d", i)
		} else if block.Height != i {
			t.Errorf("Block has incorrect height: expected %d, got %d", i, block.Height)
		}
	}

	// Add one more block
	tx := NewTransaction([]byte("Final test block"))
	_, err = bc2.AddBlock([]*Transaction{tx})
	if err != nil {
		t.Errorf("Failed to add block after reopening: %v", err)
	}

	// Verify the new height
	if bc2.GetHeight() != 4 {
		t.Errorf("Expected blockchain height to be 4 after adding block, got %d", bc2.GetHeight())
	}

	// Validate the chain
	valid, err := bc2.ValidateChain()
	if err != nil {
		t.Errorf("Chain validation failed with error: %v", err)
	}
	if !valid {
		t.Errorf("Chain validation returned invalid chain")
	}
}
