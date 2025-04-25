// Package consensus provides consensus mechanisms for the blockchain
package consensus

import (
	"crypto/sha256"
	"errors"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ConflictResolutionStrategy defines different approaches to conflict resolution
type ConflictResolutionStrategy int

const (
	TimestampBased ConflictResolutionStrategy = iota
	VectorClockBased
	EntropyBased
	ProbabilisticVoting
	HybridResolution
)

// ConflictType classifies different types of conflicts
type ConflictType int

const (
	TransactionConflict ConflictType = iota
	StateConflict
	ConsensusConflict
	CrossShardConflict
)

// ConflictDetail stores information about a conflict
type ConflictDetail struct {
	ID               string
	Type             ConflictType
	Entities         []string
	Data             map[string][]byte
	DetectedAt       time.Time
	VectorClocks     []map[string]uint64
	EntropyScores    []float64
	Timestamps       []int64
	ResolutionStatus ConflictResolutionStatus
	Resolution       string
	ResolutionProof  []byte
}

// ConflictResolutionStatus tracks the status of conflict resolution
type ConflictResolutionStatus int

const (
	ConflictDetected ConflictResolutionStatus = iota
	ConflictAnalyzing
	ConflictResolved
	ConflictDeferred
	ConflictUnresolvable
)

// AdvancedConflictResolver implements sophisticated conflict resolution
type AdvancedConflictResolver struct {
	mutex                sync.RWMutex
	consistency          *AdaptiveConsistencyOrchestrator
	activeConflicts      map[string]*ConflictDetail
	resolvedConflicts    map[string]*ConflictDetail
	strategy             ConflictResolutionStrategy
	entropyThreshold     float64
	causalityViolations  int
	resolutionStatistics map[ConflictType]int
	confidenceThreshold  float64
	maxResolutionTime    time.Duration
	maxConflictHistory   int
}

// NewAdvancedConflictResolver creates a new conflict resolver
func NewAdvancedConflictResolver(consistency *AdaptiveConsistencyOrchestrator) *AdvancedConflictResolver {
	return &AdvancedConflictResolver{
		consistency:          consistency,
		activeConflicts:      make(map[string]*ConflictDetail),
		resolvedConflicts:    make(map[string]*ConflictDetail),
		strategy:             HybridResolution,
		entropyThreshold:     0.7,
		causalityViolations:  0,
		resolutionStatistics: make(map[ConflictType]int),
		confidenceThreshold:  0.85,
		maxResolutionTime:    time.Second * 30,
		maxConflictHistory:   1000,
	}
}

// DetectConflict identifies a potential conflict between entities
func (acr *AdvancedConflictResolver) DetectConflict(
	type_ ConflictType,
	entities []string,
	data map[string][]byte,
	vectorClocks []map[string]uint64,
	timestamps []int64,
	entropyScores []float64,
) (string, error) {
	acr.mutex.Lock()
	defer acr.mutex.Unlock()

	// Generate conflict ID
	h := sha256.New()
	for _, entity := range entities {
		h.Write([]byte(entity))
	}
	for _, d := range data {
		h.Write(d)
	}
	conflictID := string(h.Sum(nil))

	// Check if this conflict is already being tracked
	if _, exists := acr.activeConflicts[conflictID]; exists {
		return conflictID, errors.New("conflict already detected and being processed")
	}

	// Create new conflict detail
	conflict := &ConflictDetail{
		ID:               conflictID,
		Type:             type_,
		Entities:         entities,
		Data:             data,
		DetectedAt:       time.Now(),
		VectorClocks:     vectorClocks,
		EntropyScores:    entropyScores,
		Timestamps:       timestamps,
		ResolutionStatus: ConflictDetected,
	}

	// Store conflict
	acr.activeConflicts[conflictID] = conflict

	// Increment conflict type statistics
	acr.resolutionStatistics[type_]++

	// Start asynchronous conflict resolution
	go acr.resolveConflict(conflictID)

	return conflictID, nil
}

// resolveConflict attempts to resolve a conflict using the chosen strategy
func (acr *AdvancedConflictResolver) resolveConflict(conflictID string) {
	acr.mutex.Lock()
	conflict, exists := acr.activeConflicts[conflictID]
	if !exists {
		acr.mutex.Unlock()
		return
	}

	// Move to analyzing state
	conflict.ResolutionStatus = ConflictAnalyzing
	acr.mutex.Unlock()

	// Apply the selected resolution strategy
	var resolutionResult string
	var confidence float64

	switch acr.strategy {
	case TimestampBased:
		resolutionResult, confidence = acr.resolveByTimestamp(conflict)
	case VectorClockBased:
		resolutionResult, confidence = acr.resolveByVectorClock(conflict)
	case EntropyBased:
		resolutionResult, confidence = acr.resolveByEntropy(conflict)
	case ProbabilisticVoting:
		resolutionResult, confidence = acr.resolveByProbabilisticVoting(conflict)
	case HybridResolution:
		resolutionResult, confidence = acr.resolveByHybridStrategy(conflict)
	}

	// Update conflict with resolution
	acr.mutex.Lock()
	defer acr.mutex.Unlock()

	if confidence >= acr.confidenceThreshold {
		// Resolution successful
		conflict.ResolutionStatus = ConflictResolved
		conflict.Resolution = resolutionResult

		// Generate a proof of resolution
		acr.generateResolutionProof(conflict)

		// Move to resolved conflicts
		acr.resolvedConflicts[conflictID] = conflict
		delete(acr.activeConflicts, conflictID)

		// Prune resolved conflicts if needed
		acr.pruneResolvedConflicts()
	} else {
		// Resolution uncertain, defer or mark as unresolvable
		if time.Since(conflict.DetectedAt) > acr.maxResolutionTime {
			conflict.ResolutionStatus = ConflictUnresolvable
		} else {
			conflict.ResolutionStatus = ConflictDeferred

			// Schedule for future resolution attempt
			go func() {
				time.Sleep(time.Second * 5)
				acr.resolveConflict(conflictID)
			}()
		}
	}
}

// resolveByTimestamp resolves conflict using timestamp ordering
func (acr *AdvancedConflictResolver) resolveByTimestamp(conflict *ConflictDetail) (string, float64) {
	if len(conflict.Timestamps) == 0 {
		return "", 0.0
	}

	// Find the earliest or latest timestamp depending on conflict type
	var selectedIndex int
	if conflict.Type == TransactionConflict {
		// For transaction conflicts, earliest timestamp wins (FIFO)
		selectedIndex = 0
		earliestTime := conflict.Timestamps[0]

		for i, ts := range conflict.Timestamps {
			if ts < earliestTime {
				earliestTime = ts
				selectedIndex = i
			}
		}
	} else {
		// For state conflicts, latest timestamp wins (last write)
		selectedIndex = 0
		latestTime := conflict.Timestamps[0]

		for i, ts := range conflict.Timestamps {
			if ts > latestTime {
				latestTime = ts
				selectedIndex = i
			}
		}
	}

	// Calculate confidence based on time difference
	confidence := 0.8 // Base confidence
	if len(conflict.Timestamps) > 1 {
		// Calculate time separation from nearest competitor
		times := make([]int64, len(conflict.Timestamps))
		copy(times, conflict.Timestamps)
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })

		var timeDiff int64
		if conflict.Type == TransactionConflict {
			// Time difference between earliest and second earliest
			if len(times) > 1 {
				timeDiff = times[1] - times[0]
			}
		} else {
			// Time difference between latest and second latest
			if len(times) > 1 {
				timeDiff = times[len(times)-1] - times[len(times)-2]
			}
		}

		// Convert to milliseconds for better scaling
		timeDiffMs := float64(timeDiff) / 1e6

		// Adjust confidence based on time difference (0.1 to 0.2 bonus)
		confidenceBonus := math.Min(0.2, timeDiffMs/1000.0)
		confidence += confidenceBonus
	}

	return conflict.Entities[selectedIndex], confidence
}

// resolveByVectorClock resolves conflict using vector clock causality
func (acr *AdvancedConflictResolver) resolveByVectorClock(conflict *ConflictDetail) (string, float64) {
	if len(conflict.VectorClocks) < 2 {
		// Not enough vector clocks for comparison
		return "", 0.0
	}

	// Determine causal relationships between vector clocks
	// For each pair of vector clocks, determine if one happened-before the other
	type CausalRelation int
	const (
		Concurrent CausalRelation = iota
		HappenedBefore
		HappenedAfter
	)

	// Track relations between all pairs
	relations := make([][]CausalRelation, len(conflict.VectorClocks))
	for i := range relations {
		relations[i] = make([]CausalRelation, len(conflict.VectorClocks))
	}

	// Compare each pair of vector clocks
	for i, vcA := range conflict.VectorClocks {
		for j, vcB := range conflict.VectorClocks {
			if i == j {
				continue
			}

			relation := Concurrent
			aBeforeB := true
			bBeforeA := true

			// Check if A happened before B
			for nodeID, versionA := range vcA {
				if versionB, exists := vcB[nodeID]; exists {
					if versionA > versionB {
						bBeforeA = false
					}
				}
			}

			// Check if B happened before A
			for nodeID, versionB := range vcB {
				if versionA, exists := vcA[nodeID]; exists {
					if versionB > versionA {
						aBeforeB = false
					}
				}
			}

			if aBeforeB && !bBeforeA {
				relation = HappenedBefore
			} else if !aBeforeB && bBeforeA {
				relation = HappenedAfter
			}

			relations[i][j] = relation
		}
	}

	// Count happened-before and happened-after relations for each clock
	scores := make([]int, len(conflict.VectorClocks))
	for i := range conflict.VectorClocks {
		for j := range conflict.VectorClocks {
			if i == j {
				continue
			}

			if relations[i][j] == HappenedAfter {
				// This clock happened after another, increment score
				scores[i]++
			}
		}
	}

	// Find the vector clock that happened after the most others
	mostRecentIndex := 0
	highestScore := scores[0]
	for i, score := range scores {
		if score > highestScore {
			highestScore = score
			mostRecentIndex = i
		}
	}

	// Calculate confidence based on causality clarity
	confidence := 0.7 // Base confidence

	// Check if we have clear causality
	hasConcurrent := false
	for i := range conflict.VectorClocks {
		for j := range conflict.VectorClocks {
			if i == j {
				continue
			}
			if relations[i][j] == Concurrent {
				hasConcurrent = true
				break
			}
		}
		if hasConcurrent {
			break
		}
	}

	if !hasConcurrent {
		// Clear causality chain, higher confidence
		confidence += 0.3
	} else {
		// Partial causality, add some confidence based on relative scores
		maxPossibleScore := len(conflict.VectorClocks) - 1
		relativeScore := float64(highestScore) / float64(maxPossibleScore)
		confidence += 0.2 * relativeScore
	}

	// If there's a concurrent event, track this as a causality violation
	if hasConcurrent {
		acr.causalityViolations++
	}

	return conflict.Entities[mostRecentIndex], confidence
}

// resolveByEntropy resolves conflict by selecting the entity with optimal entropy
func (acr *AdvancedConflictResolver) resolveByEntropy(conflict *ConflictDetail) (string, float64) {
	if len(conflict.EntropyScores) == 0 {
		return "", 0.0
	}

	// For different conflict types, optimal entropy varies
	var selectedIndex int
	var confidence float64

	switch conflict.Type {
	case TransactionConflict, CrossShardConflict:
		// For these conflicts, prefer higher entropy (more randomness)
		selectedIndex = 0
		highestEntropy := conflict.EntropyScores[0]

		for i, entropy := range conflict.EntropyScores {
			if entropy > highestEntropy {
				highestEntropy = entropy
				selectedIndex = i
			}
		}

		// Confidence based on how much higher the entropy is than threshold
		entropyDiff := highestEntropy - acr.entropyThreshold
		confidence = 0.7 + math.Min(0.3, entropyDiff)

	case StateConflict, ConsensusConflict:
		// For these conflicts, prefer entropy closest to the golden ratio (0.618)
		optimalEntropy := 0.618
		selectedIndex = 0
		closestDistance := math.Abs(conflict.EntropyScores[0] - optimalEntropy)

		for i, entropy := range conflict.EntropyScores {
			distance := math.Abs(entropy - optimalEntropy)
			if distance < closestDistance {
				closestDistance = distance
				selectedIndex = i
			}
		}

		// Confidence based on how close to optimal entropy
		confidence = 0.7 + (0.3 * (1.0 - math.Min(1.0, closestDistance*5)))
	}

	return conflict.Entities[selectedIndex], confidence
}

// resolveByProbabilisticVoting uses a randomized approach weighted by various factors
func (acr *AdvancedConflictResolver) resolveByProbabilisticVoting(conflict *ConflictDetail) (string, float64) {
	// Generate a weighted probabilistic resolution
	weights := make([]float64, len(conflict.Entities))

	// Initialize weights
	for i := range weights {
		weights[i] = 1.0
	}

	// Adjust weights by timestamps (newer gets higher weight)
	if len(conflict.Timestamps) > 0 {
		// Find min and max timestamps
		minTs := conflict.Timestamps[0]
		maxTs := conflict.Timestamps[0]
		for _, ts := range conflict.Timestamps {
			if ts < minTs {
				minTs = ts
			}
			if ts > maxTs {
				maxTs = ts
			}
		}

		tsDiff := maxTs - minTs
		if tsDiff > 0 {
			for i, ts := range conflict.Timestamps {
				// Normalize to [0,1] range
				normalizedTs := float64(ts-minTs) / float64(tsDiff)
				weights[i] *= (0.5 + normalizedTs)
			}
		}
	}

	// Adjust weights by entropy scores
	if len(conflict.EntropyScores) > 0 {
		for i, entropy := range conflict.EntropyScores {
			if entropy >= acr.entropyThreshold {
				weights[i] *= (0.8 + (entropy * 0.2))
			} else {
				weights[i] *= (0.7 - ((acr.entropyThreshold - entropy) * 0.2))
			}
		}
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight <= 0 {
		return "", 0.0
	}

	// Normalize weights
	for i := range weights {
		weights[i] /= totalWeight
	}

	// Probabilistic selection based on weights
	r := rand.Float64()
	cumulativeWeight := 0.0
	selectedIndex := 0

	for i, w := range weights {
		cumulativeWeight += w
		if r <= cumulativeWeight {
			selectedIndex = i
			break
		}
	}

	// Confidence based on selected weight
	confidence := 0.5 + (weights[selectedIndex] * 0.5)

	return conflict.Entities[selectedIndex], confidence
}

// resolveByHybridStrategy combines multiple strategies for optimal resolution
func (acr *AdvancedConflictResolver) resolveByHybridStrategy(conflict *ConflictDetail) (string, float64) {
	// Get results from each strategy
	timestampResult, tsConfidence := acr.resolveByTimestamp(conflict)
	vectorClockResult, vcConfidence := acr.resolveByVectorClock(conflict)
	entropyResult, entropyConfidence := acr.resolveByEntropy(conflict)
	probabilisticResult, probConfidence := acr.resolveByProbabilisticVoting(conflict)

	// Collect all results
	type Result struct {
		entity     string
		confidence float64
		strategy   string
	}

	results := []Result{
		{timestampResult, tsConfidence, "timestamp"},
		{vectorClockResult, vcConfidence, "vectorclock"},
		{entropyResult, entropyConfidence, "entropy"},
		{probabilisticResult, probConfidence, "probabilistic"},
	}

	// Count votes for each entity
	votes := make(map[string]float64)
	for _, result := range results {
		if result.entity != "" {
			votes[result.entity] += result.confidence
		}
	}

	// Find entity with highest weighted votes
	var selectedEntity string
	var highestVote float64

	for entity, vote := range votes {
		if vote > highestVote {
			highestVote = vote
			selectedEntity = entity
		}
	}

	// Calculate agreement factor - how many strategies agreed on this entity
	agreementCount := 0
	for _, result := range results {
		if result.entity == selectedEntity {
			agreementCount++
		}
	}

	agreementFactor := float64(agreementCount) / float64(len(results))

	// Final confidence is a combination of vote strength and agreement
	confidence := (highestVote/4.0)*0.7 + (agreementFactor * 0.3)

	return selectedEntity, confidence
}

// generateResolutionProof creates a cryptographic proof of conflict resolution
func (acr *AdvancedConflictResolver) generateResolutionProof(conflict *ConflictDetail) {
	// Create a hash of resolution details
	h := sha256.New()
	h.Write([]byte(conflict.ID))
	h.Write([]byte(conflict.Resolution))

	for _, entity := range conflict.Entities {
		h.Write([]byte(entity))
	}

	for _, data := range conflict.Data {
		h.Write(data)
	}

	h.Write([]byte(time.Now().String()))

	conflict.ResolutionProof = h.Sum(nil)
}

// pruneResolvedConflicts removes old resolved conflicts to manage memory
func (acr *AdvancedConflictResolver) pruneResolvedConflicts() {
	if len(acr.resolvedConflicts) <= acr.maxConflictHistory {
		return
	}

	// Find the oldest conflicts to remove
	conflicts := make([]*ConflictDetail, 0, len(acr.resolvedConflicts))
	for _, conflict := range acr.resolvedConflicts {
		conflicts = append(conflicts, conflict)
	}

	// Sort by detection time (oldest first)
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].DetectedAt.Before(conflicts[j].DetectedAt)
	})

	// Remove oldest conflicts
	numToRemove := len(acr.resolvedConflicts) - acr.maxConflictHistory
	for i := 0; i < numToRemove; i++ {
		delete(acr.resolvedConflicts, conflicts[i].ID)
	}
}

// GetConflictStatus returns the current status of a conflict
func (acr *AdvancedConflictResolver) GetConflictStatus(conflictID string) (*ConflictDetail, error) {
	acr.mutex.RLock()
	defer acr.mutex.RUnlock()

	// Check active conflicts
	if conflict, exists := acr.activeConflicts[conflictID]; exists {
		return conflict, nil
	}

	// Check resolved conflicts
	if conflict, exists := acr.resolvedConflicts[conflictID]; exists {
		return conflict, nil
	}

	return nil, errors.New("conflict not found")
}

// GetActiveConflictCount returns the count of currently active conflicts
func (acr *AdvancedConflictResolver) GetActiveConflictCount() int {
	acr.mutex.RLock()
	defer acr.mutex.RUnlock()

	return len(acr.activeConflicts)
}

// GetConflictSummary provides statistics about conflict resolution
func (acr *AdvancedConflictResolver) GetConflictSummary() map[string]interface{} {
	acr.mutex.RLock()
	defer acr.mutex.RUnlock()

	activeCounts := make(map[ConflictType]int)
	resolvedCounts := make(map[ConflictType]int)

	// Count active conflicts by type
	for _, conflict := range acr.activeConflicts {
		activeCounts[conflict.Type]++
	}

	// Count resolved conflicts by type
	for _, conflict := range acr.resolvedConflicts {
		resolvedCounts[conflict.Type]++
	}

	// Count unresolvable conflicts
	unresolvableCount := 0
	for _, conflict := range acr.activeConflicts {
		if conflict.ResolutionStatus == ConflictUnresolvable {
			unresolvableCount++
		}
	}

	return map[string]interface{}{
		"activeConflicts":       len(acr.activeConflicts),
		"resolvedConflicts":     len(acr.resolvedConflicts),
		"unresolvableConflicts": unresolvableCount,
		"causalityViolations":   acr.causalityViolations,
		"activeByType":          activeCounts,
		"resolvedByType":        resolvedCounts,
		"totalByType":           acr.resolutionStatistics,
	}
}

// SetStrategy changes the conflict resolution strategy
func (acr *AdvancedConflictResolver) SetStrategy(strategy ConflictResolutionStrategy) {
	acr.mutex.Lock()
	defer acr.mutex.Unlock()

	acr.strategy = strategy
}

// SetEntropyThreshold sets the threshold for entropy-based resolution
func (acr *AdvancedConflictResolver) SetEntropyThreshold(threshold float64) {
	acr.mutex.Lock()
	defer acr.mutex.Unlock()

	acr.entropyThreshold = threshold
}

// SetConfidenceThreshold sets the threshold for resolution confidence
func (acr *AdvancedConflictResolver) SetConfidenceThreshold(threshold float64) {
	acr.mutex.Lock()
	defer acr.mutex.Unlock()

	acr.confidenceThreshold = threshold
}
