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

// TestVRFProofVerification tests the generation and verification of VRF proofs
func TestVRFProofVerification(t *testing.T) {
	// Create a new BFT system
	bft := NewByzantineFaultTolerance()

	// Register a node
	nodeID := "test-node-1"
	err := bft.RegisterNode(nodeID, ValidatorNode)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Generate test data
	testData := []byte("test data for VRF proof")

	// Generate VRF proof
	proof, err := bft.GenerateVRFProof(nodeID, testData)
	if err != nil {
		t.Fatalf("Failed to generate VRF proof: %v", err)
	}

	// Verify the length of the proof (should be SHA-256 hash length)
	if len(proof) != 32 {
		t.Errorf("Expected proof length of 32 bytes, got %d bytes", len(proof))
	}

	// Verify the proof
	if !bft.VerifyVRFProof(nodeID, testData, proof) {
		t.Errorf("Failed to verify a valid VRF proof")
	}

	// Try with wrong data - should fail verification
	wrongData := []byte("wrong data")
	if bft.VerifyVRFProof(nodeID, wrongData, proof) {
		t.Errorf("Verification succeeded with wrong data")
	}

	// Try with wrong proof - should fail verification
	wrongProof := []byte("wrong proof that is definitely not correct")
	if bft.VerifyVRFProof(nodeID, testData, wrongProof) {
		t.Errorf("Verification succeeded with wrong proof")
	}
}
