package core

import (
	"bytes"
	"testing"
)

// TestNewTransaction checks if transaction creation and hashing works
func TestNewTransaction(t *testing.T) {
	data := []byte("Test Transaction Data")
	tx := NewTransaction(data)

	if tx.ID == nil {
		t.Errorf("Expected transaction ID to be set, got nil")
	}
	if !bytes.Equal(tx.Data, data) {
		t.Errorf("Expected transaction data '%s', got '%s'", data, tx.Data)
	}
	if tx.Timestamp <= 0 {
		t.Errorf("Expected positive timestamp, got %d", tx.Timestamp)
	}

	// Re-calculate hash to ensure consistency (optional sanity check)
	recalculatedHash := tx.CalculateHash()
	if !bytes.Equal(tx.ID, recalculatedHash) {
		t.Errorf("Transaction ID does not match recalculated hash")
	}
}

// TestNewBlock checks if block creation and hashing works
func TestNewBlock(t *testing.T) {
	tx1 := NewTransaction([]byte("tx1 data"))
	tx2 := NewTransaction([]byte("tx2 data"))
	transactions := []*Transaction{tx1, tx2}
	prevBlockHash := []byte("previousblockhash")
	height := int64(1)

	block := NewBlock(transactions, prevBlockHash, height)

	if block.Hash == nil {
		t.Errorf("Expected block hash to be set, got nil")
	}
	if !bytes.Equal(block.PrevBlockHash, prevBlockHash) {
		t.Errorf("Expected previous block hash '%s', got '%s'", prevBlockHash, block.PrevBlockHash)
	}
	if block.Height != height {
		t.Errorf("Expected block height %d, got %d", height, block.Height)
	}
	if len(block.Transactions) != 2 {
		t.Errorf("Expected 2 transactions in the block, got %d", len(block.Transactions))
	}
	if block.Timestamp <= 0 {
		t.Errorf("Expected positive timestamp, got %d", block.Timestamp)
	}

	// Test SetHash consistency (optional sanity check)
	initialHash := block.Hash
	block.Nonce = 12345 // Change nonce
	block.SetHash()     // Recalculate hash
	if bytes.Equal(initialHash, block.Hash) {
		t.Errorf("Expected hash to change when nonce changes, but it remained the same")
	}
}

// TestNewGenesisBlock checks genesis block creation
func TestNewGenesisBlock(t *testing.T) {
	coinbase := NewTransaction([]byte("Genesis Coinbase"))
	genesis := NewGenesisBlock(coinbase)

	if genesis.Height != 0 {
		t.Errorf("Expected genesis block height to be 0, got %d", genesis.Height)
	}
	if len(genesis.PrevBlockHash) != 0 {
		t.Errorf("Expected genesis block previous hash to be empty, got %x", genesis.PrevBlockHash)
	}
	if len(genesis.Transactions) != 1 || !bytes.Equal(genesis.Transactions[0].ID, coinbase.ID) {
		t.Errorf("Genesis block does not contain the correct coinbase transaction")
	}
	if genesis.Hash == nil {
		t.Errorf("Expected genesis block hash to be set, got nil")
	}
}
