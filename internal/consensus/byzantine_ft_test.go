package consensus

import (
	"testing"
)

// TestByzantineFaultTolerance tests the BFT system's basic functionality
func TestByzantineFaultTolerance(t *testing.T) {
	// Create a new BFT system
	bft := NewByzantineFaultTolerance()

	// Register some nodes
	nodeIDs := []string{"node-1", "node-2", "node-3", "node-4", "node-5"}
	for _, nodeID := range nodeIDs {
		err := bft.RegisterNode(nodeID, ValidatorNode)
		if err != nil {
			t.Errorf("Failed to register node %s: %v", nodeID, err)
		}
	}

	// Record some successful operations
	for i := 0; i < 3; i++ {
		bft.RecordBlockValidation(nodeIDs[i], []byte("block-hash"), true)
	}

	// Record a failure for one node
	bft.RecordBlockValidation(nodeIDs[3], []byte("block-hash"), false)

	// Generate VRF proof
	proof, err := bft.GenerateVRFProof(nodeIDs[0], []byte("data"))
	if err != nil {
		t.Errorf("Failed to generate VRF proof: %v", err)
	}

	// Verify VRF proof
	if !bft.VerifyVRFProof(nodeIDs[0], []byte("data"), proof) {
		t.Errorf("Failed to verify VRF proof")
	}

	// Check node scores
	for _, nodeID := range nodeIDs[:3] {
		score, err := bft.GetNodeScore(nodeID)
		if err != nil {
			t.Errorf("Failed to get node score for %s: %v", nodeID, err)
		}
		if score <= 0 {
			t.Errorf("Expected positive score for node %s, got %f", nodeID, score)
		}
	}

	// Test leader election
	leader, err := bft.ElectNewLeader()
	if err != nil {
		t.Errorf("Failed to elect leader: %v", err)
	}
	if leader == "" {
		t.Errorf("Elected leader should not be empty")
	}

	// Test health status
	health := bft.GetHealthStatus()
	if health["totalNodes"].(int) != 5 {
		t.Errorf("Expected 5 total nodes, got %v", health["totalNodes"])
	}

	t.Logf("BFT test passed with leader: %s", leader)
}
