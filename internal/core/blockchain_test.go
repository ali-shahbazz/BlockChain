package core

import (
	"bytes"
	"log"
	"testing"
	"time"
)

// TestNewBlockchain verifies blockchain creation with genesis block
func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain()

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
	// Set a test-specific timeout
	timeout := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		bc := NewBlockchain()
		log.Printf("Created new blockchain for testing")

		tx1 := NewTransaction([]byte("First block data"))
		tx2 := NewTransaction([]byte("More transaction data"))
		transactions := []*Transaction{tx1, tx2}

		log.Printf("Created test transactions, adding block...")
		block, err := bc.AddBlock(transactions)
		if err != nil {
			t.Errorf("Failed to add block: %v", err)
			done <- true
			return
		}

		if block.Height != 1 {
			t.Errorf("Expected block height to be 1, got %d", block.Height)
		}

		if bc.GetHeight() != 1 {
			t.Errorf("Expected blockchain height to be 1, got %d", bc.GetHeight())
		}

		// Check if we can retrieve the block
		log.Printf("Retrieving block by height...")
		retrievedBlock := bc.GetBlockByHeight(1)
		if retrievedBlock == nil {
			t.Errorf("Failed to retrieve added block")
			done <- true
			return
		}

		if !bytes.Equal(retrievedBlock.Hash, block.Hash) {
			t.Errorf("Retrieved block hash does not match added block hash")
		}

		// Check if we can retrieve by hash
		log.Printf("Retrieving block by hash...")
		hashRetrievedBlock := bc.GetBlockByHash(block.Hash)
		if hashRetrievedBlock == nil {
			t.Errorf("Failed to retrieve block by hash")
			done <- true
			return
		}

		if !bytes.Equal(hashRetrievedBlock.Hash, block.Hash) {
			t.Errorf("Hash retrieved block hash does not match")
		}

		done <- true
	}()

	// Wait for either test completion or timeout
	select {
	case <-done:
		log.Printf("TestAddBlock completed successfully")
	case <-timeout:
		t.Fatal("TestAddBlock timed out after 10 seconds")
	}
}

// TestBlockchainIterator verifies blockchain iteration
func TestBlockchainIterator(t *testing.T) {
	bc := NewBlockchain()

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
