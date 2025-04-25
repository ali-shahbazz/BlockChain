package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

// Transaction represents a basic transaction in the blockchain
type Transaction struct {
	ID        []byte
	Timestamp int64
	Data      []byte // Simple data payload for now
	// Add fields for inputs, outputs, signatures later
}

// NewTransaction creates a new transaction instance
func NewTransaction(data []byte) *Transaction {
	tx := &Transaction{
		Timestamp: time.Now().UnixNano(),
		Data:      data,
	}
	tx.ID = tx.CalculateHash()
	return tx
}

// CalculateHash computes the SHA256 hash of the transaction
func (tx *Transaction) CalculateHash() []byte {
	var encoded bytes.Buffer
	var hash [32]byte

	// Create a temporary struct excluding the ID field
	tempTx := struct {
		Timestamp int64
		Data      []byte
	}{
		Timestamp: tx.Timestamp,
		Data:      tx.Data,
	}

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tempTx) // Encode only the relevant fields
	if err != nil {
		log.Panicf("Failed to encode transaction: %v", err)
	}

	hash = sha256.Sum256(encoded.Bytes())
	return hash[:]
}

// IsCoinbase checks if the transaction is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	// A coinbase transaction typically has no inputs (in a real blockchain)
	// For our simplified implementation, we'll consider transactions with "Genesis" in their data as coinbase
	if len(tx.Data) >= 7 && bytes.HasPrefix(tx.Data, []byte("Genesis")) {
		return true
	}
	return false
}
