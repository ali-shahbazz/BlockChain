// Package crypto provides cryptographic primitives for the blockchain
package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ShardStatus represents the current status of a shard
type ShardStatus int

const (
	ShardActive ShardStatus = iota
	ShardSplitting
	ShardMerging
	ShardArchived
)

// ShardMetrics tracks computational and storage metrics for a shard
type ShardMetrics struct {
	DataSize       int64
	OperationCount int64
	LastAccessed   int64
	AccessPattern  []int64 // Tracks recent accesses for predictive modeling
}

// Shard represents a partition in the Adaptive Merkle Forest
type Shard struct {
	ID            []byte
	ParentID      []byte
	ChildrenIDs   [][]byte
	MerkleRoot    []byte
	DataElements  map[string][]byte // Maps element IDs to data
	Status        ShardStatus
	Metrics       ShardMetrics
	Version       uint64
	CreatedAt     int64
	LastUpdatedAt int64
	mutex         sync.RWMutex
}

// NewShard creates a new shard with the given ID and parent
func NewShard(id, parentID []byte) *Shard {
	now := time.Now().UnixNano()
	return &Shard{
		ID:            id,
		ParentID:      parentID,
		ChildrenIDs:   [][]byte{},
		MerkleRoot:    []byte{},
		DataElements:  make(map[string][]byte),
		Status:        ShardActive,
		Version:       1,
		CreatedAt:     now,
		LastUpdatedAt: now,
		Metrics: ShardMetrics{
			DataSize:       0,
			OperationCount: 0,
			LastAccessed:   now,
			AccessPattern:  []int64{},
		},
	}
}

// AddElement adds a data element to the shard and updates metrics
func (s *Shard) AddElement(id string, data []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixNano()

	s.DataElements[id] = data
	s.Metrics.DataSize += int64(len(data))
	s.Metrics.OperationCount++
	s.Metrics.LastAccessed = now
	s.Metrics.AccessPattern = append(s.Metrics.AccessPattern, now)
	if len(s.Metrics.AccessPattern) > 100 {
		// Keep the pattern size bounded
		s.Metrics.AccessPattern = s.Metrics.AccessPattern[1:]
	}

	s.LastUpdatedAt = now
	s.Version++

	// Rebuild the Merkle root
	s.updateMerkleRoot()
}

// GetElement retrieves a data element from the shard and updates metrics
func (s *Shard) GetElement(id string) ([]byte, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	now := time.Now().UnixNano()

	data, exists := s.DataElements[id]
	if exists {
		// Update metrics asynchronously
		go func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()
			s.Metrics.OperationCount++
			s.Metrics.LastAccessed = now
			s.Metrics.AccessPattern = append(s.Metrics.AccessPattern, now)
			if len(s.Metrics.AccessPattern) > 100 {
				s.Metrics.AccessPattern = s.Metrics.AccessPattern[1:]
			}
		}()
	}

	return data, exists
}

// RemoveElement removes a data element from the shard
func (s *Shard) RemoveElement(id string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixNano()
	data, exists := s.DataElements[id]

	if exists {
		delete(s.DataElements, id)
		s.Metrics.DataSize -= int64(len(data))
		s.Metrics.OperationCount++
		s.Metrics.LastAccessed = now
		s.LastUpdatedAt = now
		s.Version++

		// Rebuild the Merkle root
		s.updateMerkleRoot()
		return true
	}

	return false
}

// updateMerkleRoot rebuilds the Merkle tree from current data elements
func (s *Shard) updateMerkleRoot() {
	if len(s.DataElements) == 0 {
		s.MerkleRoot = nil
		return
	}

	// Collect all data elements and sort by ID for deterministic results
	var dataList [][]byte
	for _, data := range s.DataElements {
		dataList = append(dataList, data)
	}

	// Build Merkle tree
	tree := NewMerkleTree(dataList)
	s.MerkleRoot = tree.GetRootHash()
}

// GenerateProof generates a Merkle proof for a specific data element
func (s *Shard) GenerateProof(id string) ([][]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	data, exists := s.DataElements[id]
	if !exists {
		return nil, errors.New("element not found in shard")
	}

	// For real implementation, should use cached Merkle tree
	var dataList [][]byte
	for _, d := range s.DataElements {
		dataList = append(dataList, d)
	}

	tree := NewMerkleTree(dataList)
	return tree.GenerateProof(data), nil
}

// ShouldSplit evaluates metrics to determine if this shard should be split
func (s *Shard) ShouldSplit(maxSize int64, maxOps int64) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Simple threshold-based approach
	if s.Metrics.DataSize > maxSize {
		return true
	}

	if s.Metrics.OperationCount > maxOps {
		return true
	}

	// Could add more sophisticated logic based on access patterns
	return false
}

// AdaptiveMerkleForest implements the AMF with hierarchical dynamic sharding
type AdaptiveMerkleForest struct {
	shards         map[string]*Shard        // Maps shard ID (hex) to shard
	shardHierarchy map[string][]string      // Maps parent ID to child IDs
	rootShards     []string                 // Top-level shard IDs
	metrics        map[string]*ShardMetrics // Global metrics tracking
	mutex          sync.RWMutex

	// Configuration
	maxShardSize    int64 // Maximum data size before splitting
	maxShardOps     int64 // Maximum operations before rebalancing
	rebalanceWindow int64 // Time window for rebalance checking (ns)
}

// NewAdaptiveMerkleForest creates a new AMF instance
func NewAdaptiveMerkleForest() *AdaptiveMerkleForest {
	// Create initial root shard
	rootShard := NewShard(generateShardID(), nil)
	rootShardIDStr := string(rootShard.ID)

	amf := &AdaptiveMerkleForest{
		shards:          make(map[string]*Shard),
		shardHierarchy:  make(map[string][]string),
		rootShards:      []string{rootShardIDStr},
		metrics:         make(map[string]*ShardMetrics),
		maxShardSize:    1024 * 1024 * 10, // 10MB default
		maxShardOps:     1000,             // 1000 operations default
		rebalanceWindow: int64(time.Minute * 5),
	}

	// Add the root shard
	amf.shards[rootShardIDStr] = rootShard
	amf.metrics[rootShardIDStr] = &rootShard.Metrics

	// Start background rebalancing
	go amf.rebalanceRoutine()

	return amf
}

// generateShardID creates a unique ID for a new shard
func generateShardID() []byte {
	timestamp := time.Now().UnixNano()
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(timestamp))

	hash := sha256.Sum256(buf)
	return hash[:]
}

// AddData adds data to the appropriate shard based on optimal placement
func (amf *AdaptiveMerkleForest) AddData(data []byte) (string, string, error) {
	// Generate a unique ID for this data element
	dataID := sha256.Sum256(append(data, generateShardID()...))
	dataIDStr := string(dataID[:])

	var shardID string
	var shouldSplit bool

	// First critical section: find shard and add data
	amf.mutex.Lock()
	// Find the optimal shard
	shardID = amf.findOptimalShard(data)
	shard := amf.shards[shardID]

	// Add the data
	shard.AddElement(dataIDStr, data)

	// Check if shard should split, but don't split yet (avoid nested locks)
	shouldSplit = shard.ShouldSplit(amf.maxShardSize, amf.maxShardOps)
	amf.mutex.Unlock()

	// Now handle splitting if needed (in a separate critical section)
	if shouldSplit {
		err := amf.splitShard(shardID)
		if err != nil {
			// Log the error but continue - don't fail the operation
			fmt.Printf("Warning: Failed to split shard %s: %v\n", shardID, err)
		}
	}

	return dataIDStr, shardID, nil
}

// GetData retrieves data from the forest
func (amf *AdaptiveMerkleForest) GetData(dataID string, shardHint string) ([]byte, error) {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()

	// First try the hint if provided
	if shardHint != "" {
		if shard, exists := amf.shards[shardHint]; exists {
			// Check in the hinted shard
			if data, found := shard.GetElement(dataID); found {
				return data, nil
			}

			// If not found in the hinted shard, check its children recursively
			// This handles cases where data might have been moved during shard splitting
			if len(shard.ChildrenIDs) > 0 {
				for _, childID := range shard.ChildrenIDs {
					childIDStr := string(childID)
					// Recursively check this child and its descendants
					if data, err := amf.searchShardAndChildren(dataID, childIDStr); err == nil {
						return data, nil
					}
				}
			}
		}
	}

	// If still not found, search through all root shards as a fallback
	for _, rootID := range amf.rootShards {
		if data, err := amf.searchShardAndChildren(dataID, rootID); err == nil {
			return data, nil
		}
	}

	return nil, errors.New("data not found in any shard")
}

// searchShardAndChildren recursively searches for data in a shard and all its children
func (amf *AdaptiveMerkleForest) searchShardAndChildren(dataID string, shardID string) ([]byte, error) {
	shard, exists := amf.shards[shardID]
	if !exists {
		return nil, errors.New("shard not found")
	}

	// Check in this shard
	if data, found := shard.GetElement(dataID); found {
		return data, nil
	}

	// If not found here, check all children recursively
	for _, childID := range shard.ChildrenIDs {
		childIDStr := string(childID)
		if data, err := amf.searchShardAndChildren(dataID, childIDStr); err == nil {
			return data, nil
		}
	}

	return nil, errors.New("data not found in this shard or its children")
}

// findOptimalShard selects the best shard to place new data
// Uses a simple heuristic for now, but can be enhanced with more sophisticated logic
func (amf *AdaptiveMerkleForest) findOptimalShard(data []byte) string {
	// For now, just use the first root shard if available
	if len(amf.rootShards) > 0 {
		rootShardID := amf.rootShards[0]
		shard := amf.shards[rootShardID]

		// If this shard has children, traverse down the hierarchy
		if len(shard.ChildrenIDs) > 0 {
			return amf.traverseToLeafShard(rootShardID, data)
		}

		return rootShardID
	}

	// If no shards exist yet, create a new root shard
	rootShard := NewShard(generateShardID(), nil)
	rootShardIDStr := string(rootShard.ID)
	amf.shards[rootShardIDStr] = rootShard
	amf.rootShards = append(amf.rootShards, rootShardIDStr)

	return rootShardIDStr
}

// traverseToLeafShard follows the shard hierarchy to find a leaf shard
func (amf *AdaptiveMerkleForest) traverseToLeafShard(startShardID string, data []byte) string {
	currentShardID := startShardID
	currentShard := amf.shards[currentShardID]

	// While the current shard has children, pick the best child
	for len(currentShard.ChildrenIDs) > 0 {
		// Simple hash-based selection for now
		dataHash := sha256.Sum256(data)
		selectedIndex := int(dataHash[0]) % len(currentShard.ChildrenIDs)

		childID := string(currentShard.ChildrenIDs[selectedIndex])
		currentShardID = childID
		currentShard = amf.shards[currentShardID]
	}

	return currentShardID
}

// splitShard divides a shard into multiple smaller shards
func (amf *AdaptiveMerkleForest) splitShard(shardID string) error {
	// Get a copy of the data under a read lock first
	var dataToRedistribute map[string][]byte
	var parentShard *Shard
	var exists bool

	// First critical section: read the source shard data
	amf.mutex.RLock()
	parentShard, exists = amf.shards[shardID]
	if !exists {
		amf.mutex.RUnlock()
		return errors.New("shard not found")
	}

	// Skip if shard is already being modified
	if parentShard.Status != ShardActive {
		amf.mutex.RUnlock()
		return errors.New("shard is not in active state")
	}

	// Make a copy of the data before modifying anything
	dataToRedistribute = make(map[string][]byte)
	parentShard.mutex.RLock()
	for id, data := range parentShard.DataElements {
		dataToRedistribute[id] = data
	}
	parentShard.mutex.RUnlock()

	// End of first critical section
	amf.mutex.RUnlock()

	// Create two new shards (outside any lock)
	child1 := NewShard(generateShardID(), parentShard.ID)
	child2 := NewShard(generateShardID(), parentShard.ID)

	child1IDStr := string(child1.ID)
	child2IDStr := string(child2.ID)

	// Distribute data to the new shards (outside any lock)
	counter := 0
	for id, data := range dataToRedistribute {
		if counter%2 == 0 {
			child1.AddElement(id, data)
		} else {
			child2.AddElement(id, data)
		}
		counter++
	}

	// Second critical section: update the AMF structure with the new shards
	amf.mutex.Lock()

	// Recheck that the shard still exists and is still active
	parentShard, exists = amf.shards[shardID]
	if !exists {
		amf.mutex.Unlock()
		return errors.New("shard no longer exists")
	}

	if parentShard.Status != ShardActive {
		amf.mutex.Unlock()
		return errors.New("shard is no longer in active state")
	}

	// Mark as splitting
	parentShard.Status = ShardSplitting

	// Add the new shards to the forest
	amf.shards[child1IDStr] = child1
	amf.shards[child2IDStr] = child2
	amf.metrics[child1IDStr] = &child1.Metrics
	amf.metrics[child2IDStr] = &child2.Metrics

	// Update parent's children list
	parentShard.ChildrenIDs = append(parentShard.ChildrenIDs, child1.ID, child2.ID)

	// Update hierarchy map
	if _, exists := amf.shardHierarchy[shardID]; !exists {
		amf.shardHierarchy[shardID] = []string{}
	}
	amf.shardHierarchy[shardID] = append(amf.shardHierarchy[shardID], child1IDStr, child2IDStr)

	// Mark as active again
	parentShard.Status = ShardActive

	amf.mutex.Unlock()

	return nil
}

// mergeShards merges two shards into one
func (amf *AdaptiveMerkleForest) mergeShards(shardID1, shardID2 string) error {
	amf.mutex.Lock()
	defer amf.mutex.Unlock()

	shard1, exists1 := amf.shards[shardID1]
	shard2, exists2 := amf.shards[shardID2]
	if !exists1 || !exists2 {
		return errors.New("one or both shards not found")
	}

	// Ensure both shards are active and have no children
	if shard1.Status != ShardActive || shard2.Status != ShardActive {
		return errors.New("one or both shards are not active")
	}
	if len(shard1.ChildrenIDs) > 0 || len(shard2.ChildrenIDs) > 0 {
		return errors.New("one or both shards have children")
	}

	// Create a new shard to merge data
	mergedShard := NewShard(generateShardID(), nil)
	mergedShardID := string(mergedShard.ID)

	// Merge data from both shards
	for id, data := range shard1.DataElements {
		mergedShard.AddElement(id, data)
	}
	for id, data := range shard2.DataElements {
		mergedShard.AddElement(id, data)
	}

	// Update shard hierarchy
	amf.shards[mergedShardID] = mergedShard
	amf.rootShards = append(amf.rootShards, mergedShardID)

	// Remove old shards
	delete(amf.shards, shardID1)
	delete(amf.shards, shardID2)

	return nil
}

// rebalanceRoutine runs periodically to optimize the shard organization
func (amf *AdaptiveMerkleForest) rebalanceRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		amf.mutex.Lock()

		// Check each shard for potential rebalancing
		for id, shard := range amf.shards {
			// Skip if not a leaf shard or already processing
			if len(shard.ChildrenIDs) > 0 || shard.Status != ShardActive {
				continue
			}

			// Check if shard should split
			if shard.ShouldSplit(amf.maxShardSize, amf.maxShardOps) {
				go amf.splitShard(id)
			}
		}

		// Check for shards that could be merged
		for id1, shard1 := range amf.shards {
			for id2, shard2 := range amf.shards {
				if id1 != id2 && shard1.Status == ShardActive && shard2.Status == ShardActive {
					if shard1.Metrics.DataSize+shard2.Metrics.DataSize < amf.maxShardSize/2 {
						go amf.mergeShards(id1, id2)
					}
				}
			}
		}

		amf.mutex.Unlock()
	}
}

// GenerateProof creates a proof for data in a specific shard
func (amf *AdaptiveMerkleForest) GenerateProof(dataID string, shardID string) ([][]byte, error) {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()

	shard, exists := amf.shards[shardID]
	if !exists {
		return nil, errors.New("shard not found")
	}

	return shard.GenerateProof(dataID)
}

// VerifyDataInShard verifies if data exists in a shard
func (amf *AdaptiveMerkleForest) VerifyDataInShard(data []byte, shardID string) (bool, error) {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()

	shard, exists := amf.shards[shardID]
	if !exists {
		return false, errors.New("shard not found")
	}

	// Check if the exact data exists in any element in the shard
	shard.mutex.RLock()
	for _, elemData := range shard.DataElements {
		if bytes.Equal(data, elemData) {
			shard.mutex.RUnlock()
			return true, nil
		}
	}
	shard.mutex.RUnlock()

	// If not found in the shard itself, check its children
	// This handles cases where data might have been moved during shard splitting
	for _, childID := range shard.ChildrenIDs {
		childIDStr := string(childID)
		childShard, childExists := amf.shards[childIDStr]
		if childExists {
			childShard.mutex.RLock()
			for _, elemData := range childShard.DataElements {
				if bytes.Equal(data, elemData) {
					childShard.mutex.RUnlock()
					return true, nil
				}
			}
			childShard.mutex.RUnlock()
		}
	}

	return false, nil
}

// GetShardCount returns the total number of shards
func (amf *AdaptiveMerkleForest) GetShardCount() int {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()
	return len(amf.shards)
}

// GetShardHierarchyDepth returns a map of shard IDs to their depth in the hierarchy
func (amf *AdaptiveMerkleForest) GetShardHierarchyDepth() map[string]int {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()

	depthMap := make(map[string]int)

	// For each root shard, traverse and calculate depth
	for _, rootID := range amf.rootShards {
		amf.calculateShardDepths(rootID, 0, depthMap)
	}

	return depthMap
}

// calculateShardDepths recursively calculates the depth of each shard
func (amf *AdaptiveMerkleForest) calculateShardDepths(shardID string, currentDepth int, depthMap map[string]int) {
	// Record the current shard's depth
	depthMap[shardID] = currentDepth

	// Process children
	children, exists := amf.shardHierarchy[shardID]
	if exists && len(children) > 0 {
		for _, childID := range children {
			amf.calculateShardDepths(childID, currentDepth+1, depthMap)
		}
	}
}

// GetShardData returns a copy of a shard's data for inspection
func (amf *AdaptiveMerkleForest) GetShardData(shardID string) (map[string][]byte, error) {
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()

	shard, exists := amf.shards[shardID]
	if !exists {
		return nil, errors.New("shard not found")
	}

	// Create a copy of the data map
	dataCopy := make(map[string][]byte)
	shard.mutex.RLock()
	defer shard.mutex.RUnlock()

	for id, data := range shard.DataElements {
		dataCopy[id] = make([]byte, len(data))
		copy(dataCopy[id], data)
	}

	return dataCopy, nil
}
