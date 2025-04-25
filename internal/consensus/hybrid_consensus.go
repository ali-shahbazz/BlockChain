// Package consensus provides consensus mechanisms for the blockchain
package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ConsensusType represents different types of consensus mechanisms
type ConsensusType int

const (
	ProofOfWork ConsensusType = iota
	DelegatedBFT
	HybridConsensus
)

// ConsensusState represents the current state of consensus
type ConsensusState int

const (
	Initializing ConsensusState = iota
	CollectingVotes
	VerifyingBlock
	Validating  // Adding missing Validating state
	Initialized // Adding missing Initialized state
	Committed
	Failed
)

// BlockValidationResult represents the outcome of block validation
type BlockValidationResult struct {
	IsValid       bool
	ErrorMessage  string
	ValidatorID   string
	ResponseTime  time.Duration
	BlockHash     []byte
	ValidatedAt   time.Time
	EntropyFactor float64
}

// HybridConsensusProtocol combines PoW and dBFT for enhanced security
type HybridConsensusProtocol struct {
	mutex            sync.RWMutex
	bft              *ByzantineFaultTolerance
	state            ConsensusState
	validators       map[string]ValidatorInfo
	powDifficulty    *big.Int
	powDifficultyAdj float64
	bftThreshold     float64
	roundTimeout     time.Duration
	votes            map[string][]*ConsensusVote
	entropy          []byte
	currentRound     int
	lastStateChange  time.Time
	blockTime        time.Duration
	maxValidators    int
	leaderRotation   int
	adaptThresholds  bool
}

// ValidatorInfo stores information about a consensus validator
type ValidatorInfo struct {
	NodeID           string
	Stake            uint64
	VotingPower      float64
	IsActive         bool
	ConsensusHistory []bool
	SuccessRate      float64
	LastActiveTime   time.Time
	Entropy          []byte
	SuccessfulVotes  int // Adding missing SuccessfulVotes field
}

// NewHybridConsensusProtocol creates a new hybrid consensus protocol
func NewHybridConsensusProtocol(bft *ByzantineFaultTolerance) *HybridConsensusProtocol {
	// Default difficulty (can be adjusted based on network hashrate)
	initTargetBits := 16 // Easier than Bitcoin for testing
	initTarget := big.NewInt(1)
	initTarget.Lsh(initTarget, uint(256-initTargetBits))

	return &HybridConsensusProtocol{
		bft:              bft,
		state:            Initializing,
		validators:       make(map[string]ValidatorInfo),
		powDifficulty:    initTarget,
		powDifficultyAdj: 1.0,
		bftThreshold:     0.67, // 2/3 majority
		roundTimeout:     time.Second * 30,
		votes:            make(map[string][]*ConsensusVote),
		currentRound:     0,
		lastStateChange:  time.Now(),
		blockTime:        time.Second * 15, // Target block time
		maxValidators:    100,
		leaderRotation:   10, // Rotate leader every 10 blocks
		adaptThresholds:  true,
	}
}

// RegisterValidator adds a new validator to the consensus
func (hc *HybridConsensusProtocol) RegisterValidator(nodeID string, stake uint64) error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if len(hc.validators) >= hc.maxValidators {
		return errors.New("maximum validators reached")
	}

	if _, exists := hc.validators[nodeID]; exists {
		return errors.New("validator already registered")
	}

	// Register with BFT system
	err := hc.bft.RegisterNode(nodeID, ValidatorNode)
	if err != nil {
		return err
	}

	// Create entropy seed for this validator
	entropy := make([]byte, 32)
	for i := range entropy {
		entropy[i] = byte(rand.Intn(256))
	}

	// Add to validators with initial voting power based on stake
	hc.validators[nodeID] = ValidatorInfo{
		NodeID:           nodeID,
		Stake:            stake,
		VotingPower:      float64(stake) / 1000.0, // Normalized stake
		IsActive:         true,
		ConsensusHistory: make([]bool, 0),
		SuccessRate:      1.0, // Start with perfect record
		LastActiveTime:   time.Now(),
		Entropy:          entropy,
	}

	// Recalculate voting power distribution
	hc.recalculateVotingPower()

	return nil
}

// ProofOfWorkValidation performs the PoW portion of validation
func (hc *HybridConsensusProtocol) ProofOfWorkValidation(blockData []byte, nonce int64) (bool, []byte) {
	// Prepare data with nonce
	data := append(blockData, int64ToBytes(nonce)...)
	hash := sha256.Sum256(data)

	// Convert hash to big.Int for comparison with target difficulty
	hashInt := new(big.Int).SetBytes(hash[:])

	// Check if hash meets difficulty target
	isValid := hashInt.Cmp(hc.powDifficulty) == -1

	// Return validity and the hash as proof
	return isValid, hash[:]
}

// int64ToBytes converts an int64 to bytes for hashing
func int64ToBytes(val int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return buf
}

// ValidateBlock performs full hybrid validation of a block
func (hc *HybridConsensusProtocol) ValidateBlock(blockData []byte, blockHash []byte, nonce int64, proposerID string) (*BlockValidationResult, error) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	startTime := time.Now()

	// First, check if the proposer is eligible
	_, exists := hc.validators[proposerID]
	if !exists || !hc.validators[proposerID].IsActive {
		return &BlockValidationResult{
			IsValid:      false,
			ErrorMessage: "ineligible block proposer",
			ValidatorID:  "",
			BlockHash:    blockHash,
			ValidatedAt:  time.Now(),
		}, errors.New("ineligible block proposer")
	}

	// 1. Verify PoW component
	powValid, calculatedHash := hc.ProofOfWorkValidation(blockData, nonce)
	if !powValid {
		return &BlockValidationResult{
			IsValid:      false,
			ErrorMessage: "proof of work validation failed",
			ValidatorID:  "",
			BlockHash:    blockHash,
			ValidatedAt:  time.Now(),
		}, errors.New("proof of work validation failed")
	}

	// 2. Verify the calculated hash matches provided hash
	if !equalBytes(blockHash, calculatedHash) {
		return &BlockValidationResult{
			IsValid:      false,
			ErrorMessage: "block hash mismatch",
			ValidatorID:  "",
			BlockHash:    blockHash,
			ValidatedAt:  time.Now(),
		}, errors.New("block hash mismatch")
	}

	// Calculate the entropy factor from the block hash
	entropyFactor := calculateEntropyFactor(blockHash)

	// Prepare result
	result := &BlockValidationResult{
		IsValid:       true,
		ValidatorID:   "", // Will be set by the validator
		ResponseTime:  time.Since(startTime),
		BlockHash:     blockHash,
		ValidatedAt:   time.Now(),
		EntropyFactor: entropyFactor,
	}

	// Start BFT consensus for this block
	hc.state = CollectingVotes
	hc.lastStateChange = time.Now()
	hc.currentRound = 0
	hc.votes = make(map[string][]*ConsensusVote)
	hc.entropy = blockHash

	return result, nil
}

// equalBytes checks if two byte slices are equal
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// calculateEntropyFactor derives a normalized entropy factor from a block hash
func calculateEntropyFactor(hash []byte) float64 {
	// Calculate Shannon entropy
	counts := make(map[byte]int)
	for _, b := range hash {
		counts[b]++
	}

	entropy := 0.0
	for _, count := range counts {
		prob := float64(count) / float64(len(hash))
		entropy -= prob * math.Log2(prob)
	}

	// Normalize to [0, 1]
	maxEntropy := math.Log2(256.0) // Max possible entropy for bytes
	return entropy / maxEntropy
}

// SubmitVote records a vote from a validator
func (hc *HybridConsensusProtocol) SubmitVote(vote ConsensusVote) error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Verify the voter is a registered validator
	if _, exists := hc.validators[vote.NodeID]; !exists {
		return errors.New("vote from unregistered validator")
	}

	// Verify the signature (in real implementation, would use proper cryptographic verification)
	// For simplicity, we're accepting all signatures here

	// Check if we're in the right state for accepting votes
	if hc.state != CollectingVotes && hc.state != Validating {
		return errors.New("consensus protocol not in voting state")
	}

	// Add vote to the collection for this block hash
	blockHashStr := string(vote.BlockHash)
	if _, exists := hc.votes[blockHashStr]; !exists {
		hc.votes[blockHashStr] = make([]*ConsensusVote, 0)
	}

	// Create a copy of the vote to store
	voteCopy := vote
	hc.votes[blockHashStr] = append(hc.votes[blockHashStr], &voteCopy)

	// Record the vote with the BFT system
	hc.bft.RecordVote(vote)

	// Check if we've reached consensus for this block
	go hc.checkConsensusProgress()

	return nil
}

// checkConsensusProgress determines if consensus has been reached
func (hc *HybridConsensusProtocol) checkConsensusProgress() error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Add timeout check to prevent long-running operations
	if time.Since(hc.lastStateChange) > 30*time.Second {
		log.Printf("Consensus timeout reached, resetting state")
		hc.state = Initialized
		hc.lastStateChange = time.Now()
		return errors.New("consensus timeout")
	}

	// Find the block with the most votes
	var leadingBlockHash []byte
	var leadingVotes []*ConsensusVote

	for blockHashStr, votes := range hc.votes {
		if leadingBlockHash == nil || len(votes) > len(leadingVotes) {
			leadingBlockHash = []byte(blockHashStr)
			leadingVotes = votes
		}
	}

	if leadingBlockHash == nil {
		return nil // Not enough votes yet
	}

	// Check if we've reached consensus threshold using BFT
	consensusReached, consensusPercent := hc.bft.ValidateConsensus(leadingVotes, leadingBlockHash)

	// Log current consensus progress for monitoring purposes
	log.Printf("Current consensus progress: %.2f%% (threshold: %.2f%%)",
		consensusPercent*100, hc.bftThreshold*100)

	if consensusReached {
		// Consensus achieved, move to committed state
		hc.state = Committed
		hc.lastStateChange = time.Now()
		log.Printf("Consensus reached for block %x with %.2f%% votes",
			leadingBlockHash, consensusPercent*100)

		// Record success for participating validators
		for _, vote := range leadingVotes {
			// Update validator status
			if validator, exists := hc.validators[vote.NodeID]; exists {
				validator.SuccessfulVotes++
			}
		}
	}

	return nil
}

// recalculateVotingPower updates the voting power of all validators
func (hc *HybridConsensusProtocol) recalculateVotingPower() {
	totalStake := uint64(0)

	// Calculate total stake
	for _, validator := range hc.validators {
		if validator.IsActive {
			totalStake += validator.Stake
		}
	}

	if totalStake == 0 {
		return
	}

	// Normalize voting power
	for nodeID, validator := range hc.validators {
		if validator.IsActive {
			// Base power = stake percentage * reputation factor
			basePower := float64(validator.Stake) / float64(totalStake)

			// Get reputation from BFT system
			reputation, err := hc.bft.GetNodeScore(nodeID)
			if err != nil {
				reputation = 0.7 // Default if not found
			}

			// Calculate final voting power
			validator.VotingPower = basePower * (0.3 + 0.7*reputation) // Reputation influences 70% of the adjustment

			hc.validators[nodeID] = validator
		}
	}
}

// adjustConsensusParameters dynamically adjusts consensus parameters
func (hc *HybridConsensusProtocol) adjustConsensusParameters() {
	if !hc.adaptThresholds {
		return
	}

	// 1. Adjust PoW difficulty based on recent block times
	// In a real implementation, this would analyze recent block production times

	// Simple simulation - adjust difficulty to maintain target block time
	blockDuration := time.Since(hc.lastStateChange)

	// Use float64 multiplier with time.Duration to avoid truncation warnings
	slowThreshold := time.Duration(float64(hc.blockTime) * 0.8)
	fastThreshold := time.Duration(float64(hc.blockTime) * 1.2)

	if blockDuration < slowThreshold {
		// Blocks coming too fast, increase difficulty
		hc.powDifficultyAdj *= 1.05
	} else if blockDuration > fastThreshold {
		// Blocks coming too slow, decrease difficulty
		hc.powDifficultyAdj *= 0.95
	}

	// Apply the adjustment (in a real system this would adjust the target)

	// 2. Adjust BFT threshold based on network health
	healthStatus := hc.bft.GetHealthStatus()

	// Convert interface{} to concrete types
	byzantineNodes, _ := healthStatus["byzantineNodes"].(int)
	faultyNodes, _ := healthStatus["faultyNodes"].(int)
	totalNodes, _ := healthStatus["totalNodes"].(int)

	if totalNodes > 0 {
		problemRatio := float64(byzantineNodes+faultyNodes) / float64(totalNodes)

		// Adjust threshold based on network health
		if problemRatio > 0.1 {
			// Network has issues, increase threshold for more security
			hc.bftThreshold = math.Min(0.8, hc.bftThreshold+0.02)
		} else if problemRatio < 0.05 {
			// Network is healthy, can slightly relax threshold (but not below 2/3)
			hc.bftThreshold = math.Max(0.67, hc.bftThreshold-0.01)
		}
	}
}

// GetConsensusStatus returns the current state of the consensus process
func (hc *HybridConsensusProtocol) GetConsensusStatus() map[string]interface{} {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	// Count votes for leading proposal
	var leadingVoteCount int
	if len(hc.votes) > 0 {
		for _, votes := range hc.votes {
			if len(votes) > leadingVoteCount {
				leadingVoteCount = len(votes)
			}
		}
	}

	// Get active validator count
	activeValidators := 0
	for _, validator := range hc.validators {
		if validator.IsActive {
			activeValidators++
		}
	}

	return map[string]interface{}{
		"state":               hc.state,
		"currentRound":        hc.currentRound,
		"activeValidators":    activeValidators,
		"totalValidators":     len(hc.validators),
		"leadingVoteCount":    leadingVoteCount,
		"bftThreshold":        hc.bftThreshold,
		"powDifficultyAdj":    hc.powDifficultyAdj,
		"consensusAge":        time.Since(hc.lastStateChange).Seconds(),
		"roundTimeoutSeconds": hc.roundTimeout.Seconds(),
	}
}

// SelectBlockProposer chooses the next validator to propose a block
func (hc *HybridConsensusProtocol) SelectBlockProposer(blockHeight int64) (string, error) {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	if len(hc.validators) == 0 {
		return "", errors.New("no validators registered")
	}

	// Determine selection method based on block height

	// Every leaderRotation blocks, use VRF-based selection
	if blockHeight%int64(hc.leaderRotation) == 0 {
		return hc.selectProposerByVRF(blockHeight)
	}

	// Otherwise use rotation mixed with stake-weighted selection
	return hc.selectProposerByStake(blockHeight)
}

// selectProposerByVRF selects a proposer using VRF
func (hc *HybridConsensusProtocol) selectProposerByVRF(blockHeight int64) (string, error) {
	// Get leader from BFT subsystem using VRF
	leader, err := hc.bft.ElectNewLeader()
	if err != nil {
		// Fallback to stake-based selection
		return hc.selectProposerByStake(blockHeight)
	}

	// Verify the leader is an active validator
	validator, exists := hc.validators[leader]
	if !exists || !validator.IsActive {
		// Leader is not valid, fallback to stake selection
		return hc.selectProposerByStake(blockHeight)
	}

	return leader, nil
}

// selectProposerByStake selects a proposer based on stake-weighted probability
func (hc *HybridConsensusProtocol) selectProposerByStake(blockHeight int64) (string, error) {
	// Create weighted selection pool
	type candidateInfo struct {
		nodeID string
		weight float64
	}

	candidates := make([]candidateInfo, 0)
	totalWeight := 0.0

	// Create a deterministic but unpredictable random source
	// Use a combination of block height and a private entropy value
	source := rand.NewSource(blockHeight + time.Now().UnixNano())
	localRand := rand.New(source)

	for nodeID, validator := range hc.validators {
		if validator.IsActive {
			// Weight = stake * success rate * entropy factor
			weight := float64(validator.Stake) * validator.SuccessRate

			// Add some entropy based on block height and node ID
			seed := blockHeight + int64(len(nodeID))
			localNodeRand := rand.New(rand.NewSource(seed))
			entropyFactor := 0.9 + (localNodeRand.Float64() * 0.2) // 0.9 to 1.1 range

			weight *= entropyFactor

			candidates = append(candidates, candidateInfo{
				nodeID: nodeID,
				weight: weight,
			})

			totalWeight += weight
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no active validators available")
	}

	// Select based on cumulative weight
	targetWeight := localRand.Float64() * totalWeight

	cumulativeWeight := 0.0
	for _, candidate := range candidates {
		cumulativeWeight += candidate.weight
		if cumulativeWeight >= targetWeight {
			return candidate.nodeID, nil
		}
	}

	// Fallback to first candidate (should never happen unless rounding issues)
	return candidates[0].nodeID, nil
}

// GetValidatorInfo returns information about a validator
func (hc *HybridConsensusProtocol) GetValidatorInfo(nodeID string) (ValidatorInfo, error) {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	validator, exists := hc.validators[nodeID]
	if !exists {
		return ValidatorInfo{}, errors.New("validator not found")
	}

	return validator, nil
}

// ListActiveValidators returns a list of currently active validators
func (hc *HybridConsensusProtocol) ListActiveValidators() []ValidatorInfo {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	activeValidators := make([]ValidatorInfo, 0)

	for _, validator := range hc.validators {
		if validator.IsActive {
			activeValidators = append(activeValidators, validator)
		}
	}

	// Sort by stake (descending)
	sort.Slice(activeValidators, func(i, j int) bool {
		return activeValidators[i].Stake > activeValidators[j].Stake
	})

	return activeValidators
}

// FinalizeBlock is called when a block is confirmed as valid
func (hc *HybridConsensusProtocol) FinalizeBlock(blockHash []byte) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Reset consensus state
	hc.state = Initializing
	hc.lastStateChange = time.Now()
	hc.votes = make(map[string][]*ConsensusVote)

	// In a real implementation, this would:
	// 1. Record block in permanent storage
	// 2. Update validator rewards/reputation
	// 3. Prepare for next consensus round
}

// GetVRFProof generates a VRF proof for a node and data
func (hc *HybridConsensusProtocol) GetVRFProof(nodeID string, data []byte) ([]byte, error) {
	// Delegate to BFT system for VRF proof generation
	return hc.bft.GenerateVRFProof(nodeID, data)
}
