package consensus

import (
	"bytes"
	"strconv"
	"testing"
	"time"
)

// TestHybridConsensusInitialization verifies the hybrid consensus protocol initializes correctly
func TestHybridConsensusInitialization(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	if hc == nil {
		t.Fatal("Failed to create hybrid consensus protocol")
	}

	if hc.state != Initializing {
		t.Errorf("Expected initial state to be Initializing, got %v", hc.state)
	}

	if hc.bft == nil {
		t.Error("BFT reference not properly set")
	}

	if len(hc.validators) != 0 {
		t.Errorf("Expected empty validator set initially, got %d validators", len(hc.validators))
	}
}

// TestValidatorRegistration verifies validator registration
func TestValidatorRegistration(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	// Register a validator
	err := hc.RegisterValidator("validator-1", 100)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// Verify the validator was registered
	validators := hc.ListActiveValidators()
	if len(validators) != 1 {
		t.Errorf("Expected 1 active validator, got %d", len(validators))
	}

	// Try to register the same validator again (should fail)
	err = hc.RegisterValidator("validator-1", 200)
	if err == nil {
		t.Error("Expected error when registering duplicate validator, got nil")
	}

	// Register additional validators
	err = hc.RegisterValidator("validator-2", 200)
	if err != nil {
		t.Fatalf("Failed to register second validator: %v", err)
	}

	// Check validator count
	validators = hc.ListActiveValidators()
	if len(validators) != 2 {
		t.Errorf("Expected 2 active validators, got %d", len(validators))
	}
}

// TestProofOfWorkValidation tests the PoW component of validation
func TestProofOfWorkValidation(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	testData := []byte("Test block data")

	// Try with invalid nonce
	valid, _ := hc.ProofOfWorkValidation(testData, 0)
	if valid {
		t.Error("Expected PoW validation to fail with nonce 0")
	}

	// Try with various nonces to find a valid one
	var foundValid bool
	var validNonce int64
	var validHash []byte

	for nonce := int64(1); nonce < 100000; nonce++ {
		valid, hash := hc.ProofOfWorkValidation(testData, nonce)
		if valid {
			foundValid = true
			validNonce = nonce
			validHash = hash
			break
		}
	}

	if !foundValid {
		t.Skip("Could not find valid nonce in reasonable time, skipping")
	}

	// Verify that the valid nonce is consistently validated
	valid, hash := hc.ProofOfWorkValidation(testData, validNonce)
	if !valid {
		t.Error("Previously valid nonce no longer validates")
	}

	if !bytes.Equal(hash, validHash) {
		t.Error("Hash is not deterministic for the same input")
	}
}

// TestBlockProposerSelection tests the block proposer selection mechanism
func TestBlockProposerSelection(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	// Try without any validators
	_, err := hc.SelectBlockProposer(1)
	if err == nil {
		t.Error("Expected error when selecting proposer with no validators")
	}

	// Register validators
	validatorIDs := []string{"validator-1", "validator-2", "validator-3"}
	for i, id := range validatorIDs {
		err := hc.RegisterValidator(id, uint64(100*(i+1)))
		if err != nil {
			t.Fatalf("Failed to register validator %s: %v", id, err)
		}
	}

	// Select proposers for multiple blocks
	selectedProposers := make(map[string]int)
	for height := int64(1); height <= 10; height++ {
		proposer, err := hc.SelectBlockProposer(height)
		if err != nil {
			t.Fatalf("Failed to select proposer for height %d: %v", height, err)
		}

		// Verify the selected proposer is one of our validators
		found := false
		for _, id := range validatorIDs {
			if proposer == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Selected proposer %s is not a registered validator", proposer)
		}

		selectedProposers[proposer]++
	}

	// Check that there's some distribution of proposers (not guaranteed but likely)
	if len(selectedProposers) < 2 {
		t.Logf("Warning: Only selected proposers from set %v", selectedProposers)
	}
}

// TestConsensusVoting tests the voting process
func TestConsensusVoting(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	// Register validators
	for i := 1; i <= 5; i++ {
		nodeID := "validator-" + strconv.Itoa(i)
		err := hc.RegisterValidator(nodeID, uint64(100*i))
		if err != nil {
			t.Fatalf("Failed to register validator %s: %v", nodeID, err)
		}
	}

	// Setup a block and validate it
	blockData := []byte("Test block data")
	blockHash := []byte("testhash")
	proposerID := "validator-1"

	_, err := hc.ValidateBlock(blockData, blockHash, 12345, proposerID)
	if err != nil {
		t.Fatalf("Failed to validate block: %v", err)
	}

	// Create and submit votes
	vote1 := ConsensusVote{
		NodeID:       "validator-2",
		BlockHash:    blockHash,
		VoteType:     CommitVote,
		Signature:    []byte("signature"),
		Timestamp:    time.Now().UnixNano(),
		VRFProof:     []byte("vrfproof"),
		RoundNumber:  0,
		IsCommitVote: true,
	}

	err = hc.SubmitVote(vote1)
	if err != nil {
		t.Fatalf("Failed to submit vote from validator-2: %v", err)
	}

	vote2 := ConsensusVote{
		NodeID:       "validator-3",
		BlockHash:    blockHash,
		VoteType:     CommitVote,
		Signature:    []byte("signature"),
		Timestamp:    time.Now().UnixNano(),
		VRFProof:     []byte("vrfproof"),
		RoundNumber:  0,
		IsCommitVote: true,
	}

	err = hc.SubmitVote(vote2)
	if err != nil {
		t.Fatalf("Failed to submit vote from validator-3: %v", err)
	}

	// Check consensus status
	status := hc.GetConsensusStatus()
	leadingVoteCount, _ := status["leadingVoteCount"].(int)

	if leadingVoteCount < 2 {
		t.Errorf("Expected at least 2 votes for the block, got %d", leadingVoteCount)
	}

	// Submit an invalid vote (from unregistered validator)
	invalidVote := ConsensusVote{
		NodeID:       "nonexistent-validator",
		BlockHash:    blockHash,
		VoteType:     CommitVote,
		Signature:    []byte("signature"),
		Timestamp:    time.Now().UnixNano(),
		VRFProof:     []byte("vrfproof"),
		RoundNumber:  0,
		IsCommitVote: true,
	}

	err = hc.SubmitVote(invalidVote)
	if err == nil {
		t.Error("Expected error when submitting vote from unregistered validator")
	}

	// Finalize the block
	hc.FinalizeBlock(blockHash)

	// Verify state is reset
	if hc.state != Initializing {
		t.Errorf("Expected state to be reset to Initializing after finalization, got %v", hc.state)
	}
}

// TestVRFProofGeneration tests VRF proof generation and verification
func TestVRFProofGeneration(t *testing.T) {
	bft := NewByzantineFaultTolerance()
	hc := NewHybridConsensusProtocol(bft)

	// Register a validator
	nodeID := "validator-1"
	err := hc.RegisterValidator(nodeID, 100)
	if err != nil {
		t.Fatalf("Failed to register validator: %v", err)
	}

	// Generate a VRF proof
	data := []byte("test data for VRF")
	proof, err := hc.GetVRFProof(nodeID, data)

	if err != nil {
		t.Fatalf("Failed to generate VRF proof: %v", err)
	}

	if len(proof) == 0 {
		t.Error("Generated VRF proof is empty")
	}

	// Verify that the BFT system can verify the proof
	isValid := bft.VerifyVRFProof(nodeID, data, proof)
	if !isValid {
		t.Error("VRF proof verification failed")
	}

	// Check that verification fails with incorrect data
	wrongData := []byte("wrong data")
	isValid = bft.VerifyVRFProof(nodeID, wrongData, proof)
	if isValid {
		t.Error("VRF verification should fail with incorrect data")
	}
}
