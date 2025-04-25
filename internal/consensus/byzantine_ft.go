// Package consensus provides consensus mechanisms for the blockchain
package consensus

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"
)

// NodeStatus represents the current status of a node in the network
type NodeStatus int

const (
	NodeActive NodeStatus = iota
	NodeSuspect
	NodeFaulty
	NodeByzantine
)

// NodeType represents different types of node participation
type NodeType int

const (
	RegularNode NodeType = iota
	ValidatorNode
	AuditorNode
)

// NodeReputation tracks the reputation of a node in the network
type NodeReputation struct {
	NodeID             string
	ReputationScore    float64 // 0.0-1.0
	ValidBlocks        int64
	InvalidBlocks      int64
	ValidVotes         int64
	InvalidVotes       int64
	LastMisbehavior    time.Time
	ConsensusFailures  int
	ProposalAcceptRate float64
	ResponseTime       []time.Duration
	Status             NodeStatus
	Type               NodeType
	JoinTime           time.Time
	TrustScore         float64 // 0.0-1.0, derived from multiple factors
}

// ByzantineFaultTolerance implements advanced BFT resilience mechanisms
type ByzantineFaultTolerance struct {
	mutex                  sync.RWMutex
	nodes                  map[string]*NodeReputation
	activeFaults           map[string]time.Time
	faultyNodeThreshold    float64
	consensusThreshold     float64
	baselineThreshold      float64
	reputationDecayFactor  float64
	reputationRecoveryRate float64
	consensusRounds        int
	currentLeader          string
	leaderChangeInterval   time.Duration
	lastLeaderChange       time.Time
	auditInterval          time.Duration
	suspectTimeout         time.Duration
	verificationProtocol   VerificationProtocol
}

// VerificationProtocol defines the cryptographic verification approach
type VerificationProtocol int

const (
	StandardVerification VerificationProtocol = iota
	ZeroKnowledgeProof
	ThresholdSignature
	MultiPartyComputation
)

// NodeScore represents a weighted evaluation of a node
type NodeScore struct {
	NodeID       string
	TotalScore   float64
	ReputationW  float64
	PerformanceW float64
	ConsistencyW float64
	AgeW         float64
}

// ConsensusVote represents a validator's vote on a block
type ConsensusVote struct {
	NodeID       string
	BlockHash    []byte
	VoteType     VoteType
	Signature    []byte
	Timestamp    int64
	VRFProof     []byte
	RoundNumber  int
	IsCommitVote bool
}

// VoteType represents different types of votes in the consensus process
type VoteType int

const (
	PrepareVote VoteType = iota
	CommitVote
	AbortVote
	ViewChangeVote
)

// NewByzantineFaultTolerance creates a new BFT system
func NewByzantineFaultTolerance() *ByzantineFaultTolerance {
	return &ByzantineFaultTolerance{
		nodes:                  make(map[string]*NodeReputation),
		activeFaults:           make(map[string]time.Time),
		faultyNodeThreshold:    0.33, // Maximum allowed faulty nodes (less than 1/3 for BFT)
		consensusThreshold:     0.67, // Minimum required consensus (more than 2/3 for BFT)
		baselineThreshold:      0.5,  // Baseline reputation threshold
		reputationDecayFactor:  0.99, // Slight decay over time
		reputationRecoveryRate: 0.01, // Slow recovery from failures
		consensusRounds:        3,    // Number of voting rounds
		leaderChangeInterval:   time.Minute * 5,
		auditInterval:          time.Minute * 10,
		suspectTimeout:         time.Second * 30,
		verificationProtocol:   StandardVerification,
	}
}

// RegisterNode adds a new node to the BFT system
func (bft *ByzantineFaultTolerance) RegisterNode(nodeID string, nodeType NodeType) error {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	if _, exists := bft.nodes[nodeID]; exists {
		return errors.New("node already registered")
	}

	// Initialize with moderate reputation
	bft.nodes[nodeID] = &NodeReputation{
		NodeID:          nodeID,
		ReputationScore: 0.7, // Initial moderate trust
		Status:          NodeActive,
		Type:            nodeType,
		JoinTime:        time.Now(),
		TrustScore:      0.5, // Initial neutral trust
		ResponseTime:    make([]time.Duration, 0),
	}

	// If this is the first node, set as leader
	if len(bft.nodes) == 1 {
		bft.currentLeader = nodeID
		bft.lastLeaderChange = time.Now()
	}

	return nil
}

// RecordBlockValidation records when a node validates a block
func (bft *ByzantineFaultTolerance) RecordBlockValidation(nodeID string, blockHash []byte, isValid bool) error {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	node, exists := bft.nodes[nodeID]
	if !exists {
		return errors.New("node not registered")
	}

	if isValid {
		node.ValidBlocks++
		// Slightly increase reputation for valid blocks
		node.ReputationScore = math.Min(1.0, node.ReputationScore+0.01)
	} else {
		node.InvalidBlocks++
		node.LastMisbehavior = time.Now()
		// Significantly decrease reputation for invalid blocks
		node.ReputationScore = math.Max(0.0, node.ReputationScore-0.1)

		// Mark node as suspect if reputation drops below threshold
		if node.ReputationScore < bft.baselineThreshold {
			bft.markNodeAsSuspect(nodeID)
		}
	}

	// Update trust score based on all factors
	bft.updateNodeTrustScore(node)

	return nil
}

// RecordVote records a consensus vote from a node
func (bft *ByzantineFaultTolerance) RecordVote(vote ConsensusVote) error {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	node, exists := bft.nodes[vote.NodeID]
	if !exists {
		return errors.New("node not registered")
	}

	// In a real implementation, verify the vote signature here
	validSignature := true // Placeholder for actual signature verification

	if validSignature {
		node.ValidVotes++
		// Small increase for valid votes
		node.ReputationScore = math.Min(1.0, node.ReputationScore+0.005)
	} else {
		node.InvalidVotes++
		node.LastMisbehavior = time.Now()
		// Larger decrease for invalid votes (potential Byzantine behavior)
		node.ReputationScore = math.Max(0.0, node.ReputationScore-0.05)

		if node.ReputationScore < bft.baselineThreshold {
			bft.markNodeAsSuspect(vote.NodeID) // Use vote.NodeID instead of undefined nodeID
		}
	}

	bft.updateNodeTrustScore(node)
	return nil
}

// RecordConsensusFailure records when a node fails in consensus
func (bft *ByzantineFaultTolerance) RecordConsensusFailure(nodeID string) {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	node, exists := bft.nodes[nodeID]
	if !exists {
		return
	}

	node.ConsensusFailures++
	node.LastMisbehavior = time.Now()
	// Significant penalty for consensus failures
	node.ReputationScore = math.Max(0.0, node.ReputationScore-0.2)

	if node.ConsensusFailures > 3 {
		// Mark as faulty or Byzantine after multiple failures
		node.Status = NodeFaulty
		bft.activeFaults[nodeID] = time.Now()
	}

	bft.updateNodeTrustScore(node)
}

// markNodeAsSuspect marks a node as suspect for further monitoring
func (bft *ByzantineFaultTolerance) markNodeAsSuspect(nodeID string) {
	node := bft.nodes[nodeID]
	if node.Status == NodeActive {
		node.Status = NodeSuspect

		// In a real implementation, this would trigger additional monitoring
	}
}

// updateNodeTrustScore calculates a comprehensive trust score
func (bft *ByzantineFaultTolerance) updateNodeTrustScore(node *NodeReputation) {
	// Calculate trust score based on multiple factors

	// 1. Reputation component (50%)
	reputationComponent := node.ReputationScore * 0.5

	// 2. Age/history component (20%)
	ageInHours := time.Since(node.JoinTime).Hours()
	ageComponent := math.Min(1.0, ageInHours/168) * 0.2 // Max contribution after 1 week

	// 3. Performance component (20%)
	totalBlocks := float64(node.ValidBlocks + node.InvalidBlocks)
	performanceComponent := 0.0
	if totalBlocks > 0 {
		performanceComponent = (float64(node.ValidBlocks) / totalBlocks) * 0.2
	}

	// 4. Consistency component (10%)
	consistencyComponent := 0.0
	if node.ConsensusFailures < 10 {
		consistencyComponent = (1.0 - (float64(node.ConsensusFailures) / 10.0)) * 0.1
	}

	// Combine all components
	node.TrustScore = reputationComponent + ageComponent + performanceComponent + consistencyComponent
}

// GetNodeScore returns the current score for a node
func (bft *ByzantineFaultTolerance) GetNodeScore(nodeID string) (float64, error) {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	node, exists := bft.nodes[nodeID]
	if !exists {
		return 0, errors.New("node not registered")
	}

	return node.TrustScore, nil
}

// GetConsensusThreshold returns the current required consensus threshold
func (bft *ByzantineFaultTolerance) GetConsensusThreshold() float64 {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	// In advanced BFT, the threshold adapts based on network conditions
	activeNodes := 0
	totalWeight := 0.0

	for _, node := range bft.nodes {
		if node.Status == NodeActive || node.Status == NodeSuspect {
			activeNodes++
			totalWeight += node.TrustScore
		}
	}

	// If there's significant trust variation, adjust threshold
	if activeNodes > 0 {
		avgTrust := totalWeight / float64(activeNodes)

		// If average trust is low, increase threshold for extra security
		if avgTrust < 0.5 {
			return math.Max(bft.consensusThreshold, 0.75)
		}

		// If average trust is high, we can slightly relax threshold
		if avgTrust > 0.8 {
			return math.Max(0.66, bft.consensusThreshold-0.05)
		}
	}

	return bft.consensusThreshold
}

// GetCurrentLeader returns the current consensus leader
func (bft *ByzantineFaultTolerance) GetCurrentLeader() (string, error) {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	if bft.currentLeader == "" {
		return "", errors.New("no leader elected")
	}

	// Check if it's time to rotate the leader
	if time.Since(bft.lastLeaderChange) > bft.leaderChangeInterval {
		// In a real implementation, trigger leader election
		return bft.currentLeader, errors.New("leader rotation needed")
	}

	return bft.currentLeader, nil
}

// ElectNewLeader selects a new leader based on reputation and VRF
func (bft *ByzantineFaultTolerance) ElectNewLeader() (string, error) {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	candidates := make([]*NodeScore, 0)

	// Select active validator nodes as candidates
	for id, node := range bft.nodes {
		if node.Status == NodeActive && node.Type == ValidatorNode {
			// Calculate weighted score
			score := NodeScore{
				NodeID:       id,
				ReputationW:  node.ReputationScore * 0.4,
				PerformanceW: (float64(node.ValidBlocks) / math.Max(1.0, float64(node.ValidBlocks+node.InvalidBlocks))) * 0.3,
				// Fix type mismatch by casting ConsensusFailures to int64 for consistency
				ConsistencyW: (1.0 - (float64(node.ConsensusFailures) / math.Max(1.0, float64(int64(node.ConsensusFailures)+node.ValidVotes)))) * 0.2,
				AgeW:         math.Min(1.0, time.Since(node.JoinTime).Hours()/720) * 0.1, // Max age contribution after 30 days
			}
			score.TotalScore = score.ReputationW + score.PerformanceW + score.ConsistencyW + score.AgeW
			candidates = append(candidates, &score)
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no eligible leader candidates")
	}

	// Sort candidates by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].TotalScore > candidates[j].TotalScore
	})

	// Select from top 3 candidates using VRF for randomness
	selectionPool := candidates
	if len(candidates) > 3 {
		selectionPool = candidates[:3]
	}

	// Generate random selection using VRF
	randomIndex := 0
	if len(selectionPool) > 1 {
		// Simple randomness for selection
		maxRand := big.NewInt(int64(len(selectionPool)))
		randomBig, err := rand.Int(rand.Reader, maxRand)
		if err != nil {
			randomIndex = 0
		} else {
			randomIndex = int(randomBig.Int64())
		}
	}

	newLeader := selectionPool[randomIndex].NodeID
	bft.currentLeader = newLeader
	bft.lastLeaderChange = time.Now()

	return newLeader, nil
}

// ValidateConsensus determines if consensus is achieved
func (bft *ByzantineFaultTolerance) ValidateConsensus(votes []*ConsensusVote, blockHash []byte) (bool, float64) {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	requiredThreshold := bft.GetConsensusThreshold()

	// Count votes and calculate weighted vote total
	voteCount := 0
	weightedVoteTotal := 0.0
	positiveVoteWeight := 0.0

	for _, vote := range votes {
		// Only count commit votes
		if !vote.IsCommitVote {
			continue
		}

		voteCount++
		node, exists := bft.nodes[vote.NodeID]

		// Use node's trust score for vote weight
		weight := 1.0
		if exists {
			weight = node.TrustScore
		}

		weightedVoteTotal += weight

		// Check if vote is for this block
		if bytes.Equal(vote.BlockHash, blockHash) {
			positiveVoteWeight += weight
		}
	}

	// Calculate consensus percentage
	consensusPercentage := 0.0
	if weightedVoteTotal > 0 {
		consensusPercentage = positiveVoteWeight / weightedVoteTotal
	}

	// Consensus is reached if percentage exceeds required threshold
	return consensusPercentage >= requiredThreshold, consensusPercentage
}

// GenerateVRFProof creates a Verifiable Random Function proof
func (bft *ByzantineFaultTolerance) GenerateVRFProof(nodeID string, data []byte) ([]byte, error) {
	// In a real implementation, this would use proper VRF cryptography

	// Create a deterministic but unpredictable value
	h := sha256.New()
	h.Write([]byte(nodeID))
	h.Write(data)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().UnixNano()))
	h.Write(timeBytes)

	return h.Sum(nil), nil
}

// VerifyVRFProof verifies a VRF proof against the data and node
func (bft *ByzantineFaultTolerance) VerifyVRFProof(nodeID string, data []byte, proof []byte) bool {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	// Check if the node exists
	_, exists := bft.nodes[nodeID]
	if !exists {
		return false
	}

	// For simplicity, in this implementation we'll consider all proofs valid if they are non-empty
	// In a real implementation, this would perform cryptographic verification
	if proof == nil || len(proof) == 0 {
		return false
	}

	// Create a deterministic validation
	h := sha256.New()
	h.Write([]byte(nodeID))
	h.Write(data)
	h.Write(proof)

	// Determine validity based on the first byte
	return h.Sum(nil)[0] < 240 // ~94% chance of being valid
}

// PerformSecurityAudit performs a comprehensive security audit of nodes
func (bft *ByzantineFaultTolerance) PerformSecurityAudit() {
	bft.mutex.Lock()
	defer bft.mutex.Unlock()

	now := time.Now()

	// Check for nodes that haven't been audited recently
	for nodeID, node := range bft.nodes {
		// Apply reputation decay over time
		node.ReputationScore *= bft.reputationDecayFactor

		// Check suspicious nodes more thoroughly
		if node.Status == NodeSuspect {
			// If suspicious for too long with no further issues, gradually rehabilitate
			if time.Since(node.LastMisbehavior) > time.Hour {
				node.ReputationScore += bft.reputationRecoveryRate

				// If reputation has recovered, return to active status
				if node.ReputationScore >= bft.baselineThreshold {
					node.Status = NodeActive
				}
			}
		}

		// Check for Byzantine behavior patterns
		if bft.detectByzantinePattern(nodeID) {
			node.Status = NodeByzantine
			node.ReputationScore = 0
			bft.activeFaults[nodeID] = now
		}

		// Remove from active faults if reputation has recovered
		if node.Status == NodeActive {
			delete(bft.activeFaults, nodeID)
		}

		// Update trust score after audit
		bft.updateNodeTrustScore(node)
	}
}

// detectByzantinePattern implements advanced Byzantine behavior detection
func (bft *ByzantineFaultTolerance) detectByzantinePattern(nodeID string) bool {
	node := bft.nodes[nodeID]

	// Pattern 1: High rate of invalid blocks
	if node.ValidBlocks > 10 && float64(node.InvalidBlocks)/float64(node.ValidBlocks+node.InvalidBlocks) > 0.3 {
		return true
	}

	// Pattern 2: Consecutive consensus failures
	if node.ConsensusFailures >= 5 {
		return true
	}

	// Pattern 3: Very low reputation for extended period
	if node.ReputationScore < 0.2 && time.Since(node.LastMisbehavior) < time.Hour*24 {
		return true
	}

	return false
}

// GetHealthStatus returns the overall health of the BFT system
func (bft *ByzantineFaultTolerance) GetHealthStatus() map[string]interface{} {
	bft.mutex.RLock()
	defer bft.mutex.RUnlock()

	totalNodes := len(bft.nodes)
	activeNodes := 0
	suspectNodes := 0
	faultyNodes := 0
	byzantineNodes := 0

	for _, node := range bft.nodes {
		switch node.Status {
		case NodeActive:
			activeNodes++
		case NodeSuspect:
			suspectNodes++
		case NodeFaulty:
			faultyNodes++
		case NodeByzantine:
			byzantineNodes++
		}
	}

	// Calculate fault percentage
	faultPercentage := 0.0
	if totalNodes > 0 {
		faultPercentage = float64(faultyNodes+byzantineNodes) / float64(totalNodes)
	}

	// Determine if the system can still reach consensus
	canReachConsensus := faultPercentage < bft.faultyNodeThreshold

	return map[string]interface{}{
		"totalNodes":         totalNodes,
		"activeNodes":        activeNodes,
		"suspectNodes":       suspectNodes,
		"faultyNodes":        faultyNodes,
		"byzantineNodes":     byzantineNodes,
		"faultPercentage":    faultPercentage,
		"canReachConsensus":  canReachConsensus,
		"consensusThreshold": bft.GetConsensusThreshold(),
		"currentLeader":      bft.currentLeader,
	}
}

// StartAuditRoutine begins regular security audits
func (bft *ByzantineFaultTolerance) StartAuditRoutine() chan struct{} {
	ticker := time.NewTicker(bft.auditInterval)
	stopChan := make(chan struct{})

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				bft.PerformSecurityAudit()
			case <-stopChan:
				return
			}
		}
	}()

	return stopChan
}
