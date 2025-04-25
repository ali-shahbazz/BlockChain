// Package crypto provides cryptographic primitives for the blockchain
package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"
	"sort"
	"sync"
	"time"
)

// CrossShardSynchronizer manages synchronization between shards
type CrossShardSynchronizer struct {
	mutex            sync.RWMutex
	forest           *AdaptiveMerkleForest
	pendingTransfers map[string]*StateTransfer
	commitLog        []*StateCommitment
	vectorClock      map[string]uint64 // Maps shardID to version
}

// StateTransfer represents a partial state transfer between shards
type StateTransfer struct {
	ID            string
	SourceShardID string
	TargetShardID string
	Elements      map[string][]byte
	Proof         [][]byte
	RootHash      []byte
	Timestamp     int64
	Status        TransferStatus
	Signature     []byte
}

// TransferStatus represents the status of a cross-shard transfer
type TransferStatus int

const (
	TransferPending TransferStatus = iota
	TransferCommitted
	TransferAborted
)

// StateCommitment represents a cryptographic commitment to a shard state
type StateCommitment struct {
	ShardID       string
	Version       uint64
	StateRootHash []byte
	Timestamp     int64
	Signature     []byte
}

// HomomorphicCommitment provides homomorphic properties for verification
type HomomorphicCommitment struct {
	commitment     []byte
	accumulator    *CryptographicAccumulator
	vectorClock    map[string]uint64
	shardID        string
	version        uint64
	elementIndices map[string]int // Maps element IDs to positions
}

// NewCrossShardSynchronizer creates a new synchronizer for the given forest
func NewCrossShardSynchronizer(forest *AdaptiveMerkleForest) *CrossShardSynchronizer {
	return &CrossShardSynchronizer{
		forest:           forest,
		pendingTransfers: make(map[string]*StateTransfer),
		commitLog:        make([]*StateCommitment, 0),
		vectorClock:      make(map[string]uint64),
	}
}

// InitiateStateTransfer starts a transfer of elements between shards
func (css *CrossShardSynchronizer) InitiateStateTransfer(sourceID, targetID string, elementIDs []string) (*StateTransfer, error) {
	css.mutex.Lock()
	defer css.mutex.Unlock()

	// Get source shard data
	sourceData, err := css.forest.GetShardData(sourceID)
	if err != nil {
		return nil, err
	}

	// Verify target shard exists
	_, err = css.forest.GetShardData(targetID)
	if err != nil {
		return nil, err
	}

	// Prepare elements to transfer
	elements := make(map[string][]byte)
	for _, id := range elementIDs {
		if data, exists := sourceData[id]; exists {
			elements[id] = data
		}
	}

	if len(elements) == 0 {
		return nil, errors.New("no valid elements to transfer")
	}

	// Generate transfer ID
	transferID := generateTransferID(sourceID, targetID, elements)

	// Create the transfer
	transfer := &StateTransfer{
		ID:            transferID,
		SourceShardID: sourceID,
		TargetShardID: targetID,
		Elements:      elements,
		Timestamp:     time.Now().UnixNano(),
		Status:        TransferPending,
	}

	// Generate proof for the elements (a simplified version for now)
	var dataList [][]byte
	for _, data := range elements {
		dataList = append(dataList, data)
	}
	tree := NewMerkleTree(dataList)
	transfer.RootHash = tree.GetRootHash()

	// Store the pending transfer
	css.pendingTransfers[transferID] = transfer

	return transfer, nil
}

// CommitStateTransfer finalizes a state transfer between shards
func (css *CrossShardSynchronizer) CommitStateTransfer(transferID string) error {
	css.mutex.Lock()
	defer css.mutex.Unlock()

	// Find the pending transfer
	transfer, exists := css.pendingTransfers[transferID]
	if !exists {
		return errors.New("transfer not found")
	}

	if transfer.Status != TransferPending {
		return errors.New("transfer is not in pending state")
	}

	// Get the target shard
	targetShard, exists := css.forest.shards[transfer.TargetShardID]
	if !exists {
		return errors.New("target shard not found")
	}

	// Apply the transfer - add elements to target shard
	for id, data := range transfer.Elements {
		targetShard.AddElement(id, data)
	}

	// Update vector clock for both source and target shards
	// Initialize if they don't exist
	if _, exists := css.vectorClock[transfer.SourceShardID]; !exists {
		css.vectorClock[transfer.SourceShardID] = 0
	}
	if _, exists := css.vectorClock[transfer.TargetShardID]; !exists {
		css.vectorClock[transfer.TargetShardID] = 0
	}

	// Increment both versions
	css.vectorClock[transfer.SourceShardID]++
	css.vectorClock[transfer.TargetShardID]++

	// Create state commitment
	commitment := &StateCommitment{
		ShardID:       transfer.TargetShardID,
		Version:       css.vectorClock[transfer.TargetShardID],
		StateRootHash: targetShard.MerkleRoot,
		Timestamp:     time.Now().UnixNano(),
	}

	// Add to commit log
	css.commitLog = append(css.commitLog, commitment)

	// Mark transfer as committed
	transfer.Status = TransferCommitted

	return nil
}

// AbortStateTransfer cancels a pending state transfer
func (css *CrossShardSynchronizer) AbortStateTransfer(transferID string) error {
	css.mutex.Lock()
	defer css.mutex.Unlock()

	transfer, exists := css.pendingTransfers[transferID]
	if !exists {
		return errors.New("transfer not found")
	}

	if transfer.Status != TransferPending {
		return errors.New("transfer is not in pending state")
	}

	transfer.Status = TransferAborted
	return nil
}

// VerifyStateCommitment verifies the integrity of a shard's state
func (css *CrossShardSynchronizer) VerifyStateCommitment(shardID string, expectedVersion uint64) (bool, error) {
	css.mutex.RLock()
	defer css.mutex.RUnlock()

	// Get current version from vector clock
	currentVersion, exists := css.vectorClock[shardID]
	if !exists {
		return false, errors.New("shard not found in vector clock")
	}

	// Version check
	if currentVersion != expectedVersion {
		return false, nil
	}

	// Find the commitment for this version
	var commitment *StateCommitment
	for i := len(css.commitLog) - 1; i >= 0; i-- {
		if css.commitLog[i].ShardID == shardID && css.commitLog[i].Version == expectedVersion {
			commitment = css.commitLog[i]
			break
		}
	}

	if commitment == nil {
		return false, errors.New("commitment not found for version")
	}

	// Get the shard
	shard, exists := css.forest.shards[shardID]
	if !exists {
		return false, errors.New("shard not found")
	}

	// Verify the root hash matches
	return bytes.Equal(commitment.StateRootHash, shard.MerkleRoot), nil
}

// generateTransferID creates a unique ID for a state transfer
func generateTransferID(sourceID, targetID string, elements map[string][]byte) string {
	// Sort element IDs for deterministic results
	ids := make([]string, 0, len(elements))
	for id := range elements {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Concatenate all data
	var buffer bytes.Buffer
	buffer.WriteString(sourceID)
	buffer.WriteString(targetID)

	for _, id := range ids {
		buffer.WriteString(id)
		buffer.Write(elements[id])
	}

	// Add timestamp for uniqueness
	timestamp := time.Now().UnixNano()
	binary.Write(&buffer, binary.BigEndian, timestamp)

	// Hash the result
	hash := sha256.Sum256(buffer.Bytes())
	return string(hash[:])
}

// NewHomomorphicCommitment creates a new homomorphic commitment for a shard
func NewHomomorphicCommitment(shardID string, elements map[string][]byte, vectorClock map[string]uint64) (*HomomorphicCommitment, error) {
	// Create cryptographic accumulator for the elements
	acc, err := NewCryptographicAccumulator(2048)
	if err != nil {
		return nil, err
	}

	// Map positions
	elementIndices := make(map[string]int)

	// Sort element IDs for deterministic results
	ids := make([]string, 0, len(elements))
	for id := range elements {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Add elements to accumulator
	for i, id := range ids {
		acc.Add(elements[id])
		elementIndices[id] = i
	}

	// Make a deep copy of the vector clock
	vcCopy := make(map[string]uint64)
	for id, version := range vectorClock {
		vcCopy[id] = version
	}

	// Calculate version for this shard
	version, exists := vcCopy[shardID]
	if !exists {
		version = 1
	} else {
		version++
	}

	// Create the commitment
	hc := &HomomorphicCommitment{
		accumulator:    acc,
		vectorClock:    vcCopy,
		shardID:        shardID,
		version:        version,
		elementIndices: elementIndices,
	}

	// Generate the commitment value
	var buffer bytes.Buffer
	buffer.WriteString(shardID)
	binary.Write(&buffer, binary.BigEndian, version)

	// Add accumulator value
	accBytes := acc.value.Bytes()
	buffer.Write(accBytes)

	// Hash everything
	hash := sha256.Sum256(buffer.Bytes())
	hc.commitment = hash[:]

	return hc, nil
}

// VerifyElement verifies if an element is part of the commitment
func (hc *HomomorphicCommitment) VerifyElement(id string, data []byte, proof *big.Int) bool {
	// Check if element is tracked in this commitment
	_, exists := hc.elementIndices[id]
	if !exists {
		return false
	}

	// Verify with accumulator
	return hc.accumulator.VerifyMembership(data, proof)
}

// CombineWith combines this commitment with another one (homomorphic property)
func (hc *HomomorphicCommitment) CombineWith(other *HomomorphicCommitment) (*HomomorphicCommitment, error) {
	// Combine vector clocks
	combinedVC := make(map[string]uint64)
	for id, version := range hc.vectorClock {
		combinedVC[id] = version
	}

	for id, version := range other.vectorClock {
		if existing, exists := combinedVC[id]; exists {
			if version > existing {
				combinedVC[id] = version
			}
		} else {
			combinedVC[id] = version
		}
	}

	// Create a new combined commitment with empty elements
	// In a real implementation, you would combine the elements too
	combined, err := NewHomomorphicCommitment("combined", make(map[string][]byte), combinedVC)
	if err != nil {
		return nil, err
	}

	return combined, nil
}

// GetCommitment returns the commitment value
func (hc *HomomorphicCommitment) GetCommitment() []byte {
	return hc.commitment
}

// AtomicCrossShardOperation represents an atomic operation across multiple shards
type AtomicCrossShardOperation struct {
	ID             string
	Transfers      []*StateTransfer
	Status         OperationStatus
	StartTimestamp int64
	EndTimestamp   int64
	Coordinator    string
	ParticipantIDs []string
	Log            []LogEntry
}

// OperationStatus represents the status of a cross-shard operation
type OperationStatus int

const (
	OperationPreparing OperationStatus = iota
	OperationCommitting
	OperationCommitted
	OperationRollingBack
	OperationRolledBack
)

// LogEntry represents an event in the operation log
type LogEntry struct {
	Timestamp int64
	Action    string
	ShardID   string
	Success   bool
	Message   string
}

// NewAtomicCrossShardOperation creates a new atomic operation
func NewAtomicCrossShardOperation(coordinator string, participantIDs []string) *AtomicCrossShardOperation {
	// Generate a unique ID
	opID := generateOperationID(coordinator, participantIDs)

	return &AtomicCrossShardOperation{
		ID:             opID,
		Status:         OperationPreparing,
		StartTimestamp: time.Now().UnixNano(),
		Coordinator:    coordinator,
		ParticipantIDs: participantIDs,
		Transfers:      make([]*StateTransfer, 0),
		Log:            make([]LogEntry, 0),
	}
}

// AddTransfer adds a state transfer to this atomic operation
func (op *AtomicCrossShardOperation) AddTransfer(transfer *StateTransfer) {
	op.Transfers = append(op.Transfers, transfer)

	// Log the action
	op.Log = append(op.Log, LogEntry{
		Timestamp: time.Now().UnixNano(),
		Action:    "AddTransfer",
		ShardID:   transfer.SourceShardID + "->" + transfer.TargetShardID,
		Success:   true,
		Message:   "Added transfer to operation",
	})
}

// Prepare prepares all shards for the operation
func (op *AtomicCrossShardOperation) Prepare() bool {
	op.Status = OperationPreparing

	// In a real implementation, you would contact all participants
	// and ensure they are ready to commit

	// For this example, we'll just simulate success
	allPrepared := true

	for _, id := range op.ParticipantIDs {
		success := true // Simulated response

		op.Log = append(op.Log, LogEntry{
			Timestamp: time.Now().UnixNano(),
			Action:    "Prepare",
			ShardID:   id,
			Success:   success,
			Message:   "Prepared shard for operation",
		})

		if !success {
			allPrepared = false
		}
	}

	return allPrepared
}

// Commit attempts to commit all transfers atomically
func (op *AtomicCrossShardOperation) Commit(synchronizer *CrossShardSynchronizer) error {
	if op.Status != OperationPreparing {
		return errors.New("operation not in prepared state")
	}

	op.Status = OperationCommitting

	// Commit each transfer
	for _, transfer := range op.Transfers {
		err := synchronizer.CommitStateTransfer(transfer.ID)
		success := err == nil

		op.Log = append(op.Log, LogEntry{
			Timestamp: time.Now().UnixNano(),
			Action:    "Commit",
			ShardID:   transfer.TargetShardID,
			Success:   success,
			Message: func() string {
				if success {
					return "Transfer committed"
				}
				return err.Error()
			}(),
		})

		if !success {
			// If any commit fails, roll back all previous commits
			op.Rollback(synchronizer)
			return err
		}
	}

	op.Status = OperationCommitted
	op.EndTimestamp = time.Now().UnixNano()
	return nil
}

// Rollback rolls back all transfers in this operation
func (op *AtomicCrossShardOperation) Rollback(synchronizer *CrossShardSynchronizer) {
	op.Status = OperationRollingBack

	// In a real system, you would implement a complex rollback mechanism
	// For this example, we'll just mark all pending transfers as aborted

	for _, transfer := range op.Transfers {
		if transfer.Status == TransferPending {
			err := synchronizer.AbortStateTransfer(transfer.ID)
			success := err == nil

			op.Log = append(op.Log, LogEntry{
				Timestamp: time.Now().UnixNano(),
				Action:    "Rollback",
				ShardID:   transfer.TargetShardID,
				Success:   success,
				Message: func() string {
					if success {
						return "Transfer aborted"
					}
					return err.Error()
				}(),
			})
		}
	}

	op.Status = OperationRolledBack
	op.EndTimestamp = time.Now().UnixNano()
}

// generateOperationID creates a unique ID for a cross-shard operation
func generateOperationID(coordinator string, participants []string) string {
	var buffer bytes.Buffer
	buffer.WriteString(coordinator)

	// Sort participant IDs for deterministic results
	sort.Strings(participants)
	for _, id := range participants {
		buffer.WriteString(id)
	}

	// Add timestamp for uniqueness
	timestamp := time.Now().UnixNano()
	binary.Write(&buffer, binary.BigEndian, timestamp)

	// Hash the result
	hash := sha256.Sum256(buffer.Bytes())
	return string(hash[:])
}
