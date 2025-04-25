package core

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"
)

// TestNewBlockchain verifies blockchain creation with genesis block
func TestNewBlockchain(t *testing.T) {
	// Create a temporary directory for this test
	tempDir, err := os.MkdirTemp("", "blockchain-new-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a blockchain with a clean directory
	bc := NewBlockchainWithPath(tempDir)

	if bc.GetHeight() != 0 {
		t.Errorf("Expected genesis block height to be 0, got %d", bc.GetHeight())
	}

	genesisBlock := bc.GetBlockByHeight(0)
	if genesisBlock == nil {
		t.Errorf("Failed to retrieve genesis block")
		return
	}

	if len(genesisBlock.PrevBlockHash) != 0 {
		t.Errorf("Expected genesis block previous hash to be empty")
	}
}

// TestAddBlock verifies adding blocks to the blockchain
func TestAddBlock(t *testing.T) {
	// Create a temporary directory for this test
	tempDir, err := os.MkdirTemp("", "blockchain-add-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set a test-specific timeout
	timeout := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		// Create a blockchain with the temporary directory
		bc := NewBlockchainWithPath(tempDir)
		log.Println("Created new blockchain for testing")

		// Create some test transactions
		tx1 := NewTransaction([]byte("First transaction data"))
		tx2 := NewTransaction([]byte("Second transaction data"))
		transactions := []*Transaction{tx1, tx2}
		log.Println("Created test transactions, adding block...")

		// Add a block with these transactions
		block, err := bc.AddBlock(transactions)
		if err != nil {
			t.Errorf("Failed to add block: %v", err)
			done <- true
			return
		}

		// Check if we can retrieve the block
		log.Println("Retrieving block by height...")
		retrievedByHeight := bc.GetBlockByHeight(1)
		if retrievedByHeight == nil {
			t.Error("Failed to retrieve block by height")
			done <- true
			return
		}

		// Check if hashes match
		if !bytes.Equal(retrievedByHeight.Hash, block.Hash) {
			t.Errorf("Retrieved block hash doesn't match: expected %x, got %x", block.Hash, retrievedByHeight.Hash)
		}

		// Check if we can retrieve by hash
		log.Println("Retrieving block by hash...")
		retrievedByHash := bc.GetBlockByHash(block.Hash)
		if retrievedByHash == nil {
			t.Error("Failed to retrieve block by hash")
			done <- true
			return
		}

		// Check if the height matches
		if retrievedByHash.Height != 1 {
			t.Errorf("Retrieved block has incorrect height: expected 1, got %d", retrievedByHash.Height)
		}

		log.Println("TestAddBlock completed successfully")
		done <- true
	}()

	// Wait for either test completion or timeout
	select {
	case <-done:
		// Test completed
	case <-timeout:
		t.Fatal("Test timed out after 10 seconds")
	}
}

// TestBlockchainIterator verifies blockchain iteration
func TestBlockchainIterator(t *testing.T) {
	// Create a temporary directory for this test
	tempDir, err := os.MkdirTemp("", "blockchain-iterator-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	bc := NewBlockchainWithPath(tempDir)

	// Add a few blocks
	for i := 0; i < 3; i++ {
		tx := NewTransaction([]byte("Block data"))
		_, err := bc.AddBlock([]*Transaction{tx})
		if err != nil {
			t.Errorf("Failed to add block %d: %v", i+1, err)
			return
		}
	}

	// Blockchain should have 4 blocks (genesis + 3 added)
	if bc.GetHeight() != 3 {
		t.Errorf("Expected blockchain height to be 3, got %d", bc.GetHeight())
	}

	// Iterate and count
	iterator := bc.Iterator()
	count := 0
	heights := []int64{}

	for {
		block := iterator.Next()
		if block == nil {
			break
		}
		count++
		heights = append(heights, block.Height)
	}

	if count != 4 {
		t.Errorf("Expected to iterate through 4 blocks, got %d", count)
	}

	// Heights should be in reverse order (3, 2, 1, 0)
	for i, h := range heights {
		expected := int64(3 - i)
		if h != expected {
			t.Errorf("Expected height %d at position %d, got %d", expected, i, h)
		}
	}
}
