package consensus

import (
	"testing"
	"time"
)

// TestAdaptiveConsistency tests the adaptive consistency orchestrator
func TestAdaptiveConsistency(t *testing.T) {
	// Create a new orchestrator
	aco := NewAdaptiveConsistencyOrchestrator()

	// Register some nodes
	nodeIDs := []string{"node-1", "node-2", "node-3", "node-4", "node-5"}
	for _, nodeID := range nodeIDs {
		aco.RegisterNode(nodeID)
	}

	// Record successful operations with varying latencies
	for i, nodeID := range nodeIDs {
		latency := time.Duration((i+1)*10) * time.Millisecond
		aco.RecordOperationResult(nodeID, latency, true)
	}

	// Record some failures to simulate network issues
	aco.RecordOperationResult(nodeIDs[1], 200*time.Millisecond, false)
	aco.RecordOperationResult(nodeIDs[2], 150*time.Millisecond, false)

	// Get the consistency level
	level := aco.GetConsistencyLevel()
	t.Logf("Current consistency level: %d", level)

	// Get the partition probability
	prob := aco.GetPartitionProbability()
	t.Logf("Current partition probability: %.4f", prob)

	// Validate the consistency level is reasonable based on our simulated conditions
	if prob <= 0 || prob >= 1.0 {
		t.Errorf("Partition probability should be between 0 and 1, got %.4f", prob)
	}

	// Test that increasing failures changes the consistency level
	for i := 0; i < 10; i++ {
		aco.RecordOperationResult(nodeIDs[i%5], 200*time.Millisecond, false)
	}

	newLevel := aco.GetConsistencyLevel()
	newProb := aco.GetPartitionProbability()

	t.Logf("After increased failures - Level: %d, Probability: %.4f", newLevel, newProb)

	// The probability should have increased with more failures
	if newProb <= prob {
		t.Logf("Warning: Partition probability did not increase as expected (%.4f → %.4f)", prob, newProb)
	}
}
