package crypto

import (
	"testing"
)

// TestAdaptiveMerkleForest tests the basic functionality of the AMF
func TestAdaptiveMerkleForest(t *testing.T) {
	// Create a new forest
	forest := NewAdaptiveMerkleForest()

	// Verify initial state
	if count := forest.GetShardCount(); count != 1 {
		t.Errorf("Expected initial shard count to be 1, got %d", count)
	}

	// Add some data
	data1 := []byte("Test data 1")
	dataID1, shardID1, err := forest.AddData(data1)
	if err != nil {
		t.Errorf("Failed to add data: %v", err)
	}

	// Retrieve the data
	retrievedData, err := forest.GetData(dataID1, shardID1)
	if err != nil {
		t.Errorf("Failed to retrieve data: %v", err)
	}

	// Verify data integrity
	if string(retrievedData) != string(data1) {
		t.Errorf("Retrieved data doesn't match original. Expected %s, got %s",
			string(data1), string(retrievedData))
	}

	// Test shard depth calculation
	depthMap := forest.GetShardHierarchyDepth()
	if len(depthMap) == 0 {
		t.Errorf("Expected depth map to have at least one entry")
	}

	t.Logf("AMF test passed with %d shards", forest.GetShardCount())
}
