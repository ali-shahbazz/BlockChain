// Package consensus provides CAP theorem dynamic optimization
package consensus

import (
	"math"
	"sync"
	"time"
)

// ConsistencyLevel represents different levels of consistency
type ConsistencyLevel int

const (
	StrongConsistency ConsistencyLevel = iota
	CausalConsistency
	SessionConsistency
	EventualConsistency
)

// PartitionProbability indicates the likelihood of network partitions
type PartitionProbability float64

const (
	LowPartitionProbability    PartitionProbability = 0.0
	MediumPartitionProbability PartitionProbability = 0.5
	HighPartitionProbability   PartitionProbability = 0.9
)

// NetworkStats tracks statistics about network performance
type NetworkStats struct {
	LatencyHistory     []time.Duration
	FailureHistory     []bool
	LastSyncTime       time.Time
	ResponseTimeWindow []time.Duration
	NodeAvailability   map[string]float64
	PartitionEvents    []time.Time
}

// AdaptiveConsistencyOrchestrator dynamically adjusts consistency levels
type AdaptiveConsistencyOrchestrator struct {
	mutex              sync.RWMutex
	currentConsistency ConsistencyLevel
	networkStats       NetworkStats
	nodeTelemetry      map[string]*NodeTelemetry
	// Configuration
	adaptationInterval time.Duration
	nodeTimeoutBase    time.Duration
	maxRetries         int
}

// NodeTelemetry tracks performance metrics for a specific node
type NodeTelemetry struct {
	NodeID          string
	LatencyHistory  []time.Duration
	FailureRate     float64
	LastConnected   time.Time
	SuccessfulOps   int
	FailedOps       int
	DynamicTimeout  time.Duration
	AdaptiveRetries int
	VectorClock     map[string]uint64
}

// NewAdaptiveConsistencyOrchestrator creates a new consistency orchestrator
func NewAdaptiveConsistencyOrchestrator() *AdaptiveConsistencyOrchestrator {
	aco := &AdaptiveConsistencyOrchestrator{
		currentConsistency: StrongConsistency, // Start with strong consistency
		networkStats: NetworkStats{
			LatencyHistory:     make([]time.Duration, 0),
			FailureHistory:     make([]bool, 0),
			LastSyncTime:       time.Now(),
			ResponseTimeWindow: make([]time.Duration, 0),
			NodeAvailability:   make(map[string]float64),
			PartitionEvents:    make([]time.Time, 0),
		},
		nodeTelemetry:      make(map[string]*NodeTelemetry),
		adaptationInterval: 5 * time.Second,
		nodeTimeoutBase:    500 * time.Millisecond,
		maxRetries:         3,
	}

	// Start background monitoring
	go aco.monitorNetworkConditions()

	return aco
}

// RegisterNode adds a new node for monitoring
func (aco *AdaptiveConsistencyOrchestrator) RegisterNode(nodeID string) {
	aco.mutex.Lock()
	defer aco.mutex.Unlock()

	aco.nodeTelemetry[nodeID] = &NodeTelemetry{
		NodeID:          nodeID,
		LatencyHistory:  make([]time.Duration, 0),
		LastConnected:   time.Now(),
		DynamicTimeout:  aco.nodeTimeoutBase,
		AdaptiveRetries: aco.maxRetries,
		VectorClock:     make(map[string]uint64),
	}

	aco.networkStats.NodeAvailability[nodeID] = 1.0 // Assume initially available
}

// RecordOperationResult records the result of an operation with a node
func (aco *AdaptiveConsistencyOrchestrator) RecordOperationResult(nodeID string, latency time.Duration, success bool) {
	aco.mutex.Lock()
	defer aco.mutex.Unlock()

	// Update node telemetry
	node, exists := aco.nodeTelemetry[nodeID]
	if !exists {
		// Auto-register new nodes
		node = &NodeTelemetry{
			NodeID:          nodeID,
			LatencyHistory:  make([]time.Duration, 0),
			LastConnected:   time.Now(),
			DynamicTimeout:  aco.nodeTimeoutBase,
			AdaptiveRetries: aco.maxRetries,
			VectorClock:     make(map[string]uint64),
		}
		aco.nodeTelemetry[nodeID] = node
		aco.networkStats.NodeAvailability[nodeID] = 1.0
	}

	// Update telemetry data
	if success {
		node.SuccessfulOps++
		node.LastConnected = time.Now()
		node.LatencyHistory = append(node.LatencyHistory, latency)

		// Keep history bounded
		if len(node.LatencyHistory) > 100 {
			node.LatencyHistory = node.LatencyHistory[1:]
		}
	} else {
		node.FailedOps++
	}

	// Update node failure rate
	totalOps := node.SuccessfulOps + node.FailedOps
	if totalOps > 0 {
		node.FailureRate = float64(node.FailedOps) / float64(totalOps)
	}

	// Update global network stats
	aco.networkStats.LatencyHistory = append(aco.networkStats.LatencyHistory, latency)
	if len(aco.networkStats.LatencyHistory) > 1000 {
		aco.networkStats.LatencyHistory = aco.networkStats.LatencyHistory[1:]
	}

	aco.networkStats.FailureHistory = append(aco.networkStats.FailureHistory, !success)
	if len(aco.networkStats.FailureHistory) > 1000 {
		aco.networkStats.FailureHistory = aco.networkStats.FailureHistory[1:]
	}

	aco.networkStats.ResponseTimeWindow = append(aco.networkStats.ResponseTimeWindow, latency)
	if len(aco.networkStats.ResponseTimeWindow) > 50 {
		aco.networkStats.ResponseTimeWindow = aco.networkStats.ResponseTimeWindow[1:]
	}

	// Update node availability
	aco.updateNodeAvailability(nodeID, success)

	// Adjust timeout and retries for this node
	aco.adjustNodeParameters(node)
}

// updateNodeAvailability updates the availability metric for a node
func (aco *AdaptiveConsistencyOrchestrator) updateNodeAvailability(nodeID string, success bool) {
	// Get current availability
	currentAvailability := aco.networkStats.NodeAvailability[nodeID]

	// Update with exponential moving average
	alpha := 0.2 // Weight for new observation
	successValue := 1.0
	if !success {
		successValue = 0.0
	}

	// Calculate new availability
	newAvailability := alpha*successValue + (1-alpha)*currentAvailability
	aco.networkStats.NodeAvailability[nodeID] = newAvailability
}

// adjustNodeParameters dynamically adjusts timeout and retry parameters
func (aco *AdaptiveConsistencyOrchestrator) adjustNodeParameters(node *NodeTelemetry) {
	// Adjust timeout based on latency history
	if len(node.LatencyHistory) > 0 {
		// Calculate mean and standard deviation
		sum := time.Duration(0)
		for _, latency := range node.LatencyHistory {
			sum += latency
		}
		mean := sum / time.Duration(len(node.LatencyHistory))

		varianceSum := float64(0)
		for _, latency := range node.LatencyHistory {
			diff := float64(latency - mean)
			varianceSum += diff * diff
		}
		stdDev := time.Duration(math.Sqrt(varianceSum / float64(len(node.LatencyHistory))))

		// Set timeout to mean + 2*stdDev (covers ~95% of cases)
		node.DynamicTimeout = mean + 2*stdDev

		// Add minimum floor
		if node.DynamicTimeout < aco.nodeTimeoutBase {
			node.DynamicTimeout = aco.nodeTimeoutBase
		}
	}

	// Adjust retry count based on failure rate
	failureRateThresholds := []float64{0.1, 0.3, 0.5}
	retryValues := []int{1, 2, 3, 4} // Different retry values based on failure rate

	// Set retry value based on failure rate
	if node.FailureRate < failureRateThresholds[0] {
		node.AdaptiveRetries = retryValues[0]
	} else if node.FailureRate < failureRateThresholds[1] {
		node.AdaptiveRetries = retryValues[1]
	} else if node.FailureRate < failureRateThresholds[2] {
		node.AdaptiveRetries = retryValues[2]
	} else {
		node.AdaptiveRetries = retryValues[3]
	}
}

// GetConsistencyLevel returns the current consistency level
func (aco *AdaptiveConsistencyOrchestrator) GetConsistencyLevel() ConsistencyLevel {
	aco.mutex.RLock()
	defer aco.mutex.RUnlock()
	return aco.currentConsistency
}

// GetNodeParameters returns the timeout and retry count for a node
func (aco *AdaptiveConsistencyOrchestrator) GetNodeParameters(nodeID string) (time.Duration, int, error) {
	aco.mutex.RLock()
	defer aco.mutex.RUnlock()

	node, exists := aco.nodeTelemetry[nodeID]
	if !exists {
		return aco.nodeTimeoutBase, aco.maxRetries, nil
	}

	return node.DynamicTimeout, node.AdaptiveRetries, nil
}

// GetPartitionProbability estimates the probability of network partitions
func (aco *AdaptiveConsistencyOrchestrator) GetPartitionProbability() PartitionProbability {
	aco.mutex.RLock()
	defer aco.mutex.RUnlock()

	// Use multiple factors to predict partition probability

	// Factor 1: Recent failure rate
	recentFailureRate := aco.calculateRecentFailureRate()

	// Factor 2: Latency volatility
	latencyVolatility := aco.calculateLatencyVolatility()

	// Factor 3: Node availability variance
	availabilityVariance := aco.calculateAvailabilityVariance()

	// Combine factors with weights
	combinedProbability := 0.4*recentFailureRate + 0.3*latencyVolatility + 0.3*availabilityVariance

	return PartitionProbability(combinedProbability)
}

// calculateRecentFailureRate calculates the rate of failures in recent operations
func (aco *AdaptiveConsistencyOrchestrator) calculateRecentFailureRate() float64 {
	if len(aco.networkStats.FailureHistory) == 0 {
		return 0.0
	}

	// Focus on more recent failures (last 20%)
	startIdx := int(float64(len(aco.networkStats.FailureHistory)) * 0.8)
	if startIdx >= len(aco.networkStats.FailureHistory) {
		startIdx = 0
	}

	recentFailures := aco.networkStats.FailureHistory[startIdx:]
	failureCount := 0
	for _, failed := range recentFailures {
		if failed {
			failureCount++
		}
	}

	return float64(failureCount) / float64(len(recentFailures))
}

// calculateLatencyVolatility measures how unstable the network latency is
func (aco *AdaptiveConsistencyOrchestrator) calculateLatencyVolatility() float64 {
	if len(aco.networkStats.LatencyHistory) < 2 {
		return 0.0
	}

	// Calculate standard deviation / mean (coefficient of variation)
	sum := time.Duration(0)
	for _, latency := range aco.networkStats.ResponseTimeWindow {
		sum += latency
	}

	mean := float64(sum) / float64(len(aco.networkStats.ResponseTimeWindow))
	if mean == 0 {
		return 0.0
	}

	varianceSum := 0.0
	for _, latency := range aco.networkStats.ResponseTimeWindow {
		diff := float64(latency) - mean
		varianceSum += diff * diff
	}

	stdDev := math.Sqrt(varianceSum / float64(len(aco.networkStats.ResponseTimeWindow)))

	// Coefficient of variation, normalized to [0,1]
	cv := stdDev / mean
	normalizedCV := math.Min(cv/2.0, 1.0) // Anything over 200% variation is considered maximum volatility

	return normalizedCV
}

// calculateAvailabilityVariance measures how different node availabilities are
func (aco *AdaptiveConsistencyOrchestrator) calculateAvailabilityVariance() float64 {
	if len(aco.networkStats.NodeAvailability) < 2 {
		return 0.0
	}

	// Calculate availability variance (high variance suggests partition)
	sum := 0.0
	for _, availability := range aco.networkStats.NodeAvailability {
		sum += availability
	}

	mean := sum / float64(len(aco.networkStats.NodeAvailability))

	varianceSum := 0.0
	for _, availability := range aco.networkStats.NodeAvailability {
		diff := availability - mean
		varianceSum += diff * diff
	}

	variance := varianceSum / float64(len(aco.networkStats.NodeAvailability))

	// Normalize to [0,1]
	normalizedVariance := math.Min(variance*4.0, 1.0) // Scale up since variance is typically small

	return normalizedVariance
}

// monitorNetworkConditions periodically adjusts consistency level based on network conditions
func (aco *AdaptiveConsistencyOrchestrator) monitorNetworkConditions() {
	ticker := time.NewTicker(aco.adaptationInterval)
	defer ticker.Stop()

	for range ticker.C {
		aco.adaptConsistencyLevel()
	}
}

// adaptConsistencyLevel changes consistency based on network conditions
func (aco *AdaptiveConsistencyOrchestrator) adaptConsistencyLevel() {
	aco.mutex.Lock()
	defer aco.mutex.Unlock()

	// Get current partition probability
	partitionProb := aco.GetPartitionProbability()

	// Adjust consistency level based on partition probability
	if partitionProb < 0.2 {
		// Low probability of partition, can use strong consistency
		aco.currentConsistency = StrongConsistency
	} else if partitionProb < 0.5 {
		// Medium probability, use causal consistency
		aco.currentConsistency = CausalConsistency
	} else if partitionProb < 0.8 {
		// Higher probability, use session consistency
		aco.currentConsistency = SessionConsistency
	} else {
		// Very high probability of partition, use eventual consistency
		aco.currentConsistency = EventualConsistency
	}

	// Record a partition event if probability is very high
	if partitionProb > 0.9 && len(aco.networkStats.PartitionEvents) == 0 ||
		time.Since(aco.networkStats.PartitionEvents[len(aco.networkStats.PartitionEvents)-1]) > time.Minute {
		aco.networkStats.PartitionEvents = append(aco.networkStats.PartitionEvents, time.Now())
	}
}

// MultiDimensionalConsistencyParams stores multiple consistency parameters
type MultiDimensionalConsistencyParams struct {
	Level             ConsistencyLevel
	ReadQuorum        int
	WriteQuorum       int
	ConsistencyWindow time.Duration
	MaxStaleness      time.Duration
	VectorClock       map[string]uint64
	NodeWeights       map[string]float64
}

// GetConsistencyParams returns consistency parameters for operations
func (aco *AdaptiveConsistencyOrchestrator) GetConsistencyParams(nodes []string) MultiDimensionalConsistencyParams {
	aco.mutex.RLock()
	defer aco.mutex.RUnlock()

	// Get total node count for quorum calculations
	totalNodes := len(nodes)
	if totalNodes == 0 {
		totalNodes = len(aco.nodeTelemetry)
	}

	params := MultiDimensionalConsistencyParams{
		Level:       aco.currentConsistency,
		VectorClock: make(map[string]uint64),
		NodeWeights: make(map[string]float64),
	}

	// Set parameters based on consistency level
	switch aco.currentConsistency {
	case StrongConsistency:
		// Strict quorums for strong consistency
		params.ReadQuorum = (totalNodes / 2) + 1
		params.WriteQuorum = (totalNodes / 2) + 1
		params.ConsistencyWindow = 0
		params.MaxStaleness = 0

	case CausalConsistency:
		// Causal consistency with vector clocks
		params.ReadQuorum = (totalNodes / 3) + 1
		params.WriteQuorum = (totalNodes / 3) + 1
		params.ConsistencyWindow = 100 * time.Millisecond
		params.MaxStaleness = 500 * time.Millisecond

		// Merge vector clocks from all nodes
		for _, nodeID := range nodes {
			if node, exists := aco.nodeTelemetry[nodeID]; exists {
				for id, version := range node.VectorClock {
					if current, ok := params.VectorClock[id]; !ok || version > current {
						params.VectorClock[id] = version
					}
				}
			}
		}

	case SessionConsistency:
		// More relaxed quorums
		params.ReadQuorum = totalNodes / 3
		params.WriteQuorum = totalNodes / 3
		params.ConsistencyWindow = 500 * time.Millisecond
		params.MaxStaleness = 2 * time.Second

	case EventualConsistency:
		// Minimal requirements for availability
		params.ReadQuorum = 1
		params.WriteQuorum = 1
		params.ConsistencyWindow = 1 * time.Second
		params.MaxStaleness = 10 * time.Second
	}

	// Calculate node weights based on availability and latency
	for _, nodeID := range nodes {
		weight := 1.0
		if avail, exists := aco.networkStats.NodeAvailability[nodeID]; exists {
			weight *= avail
		}

		if node, exists := aco.nodeTelemetry[nodeID]; exists && len(node.LatencyHistory) > 0 {
			// Lower weight for nodes with high latency
			avgLatency := time.Duration(0)
			for _, lat := range node.LatencyHistory {
				avgLatency += lat
			}
			avgLatency /= time.Duration(len(node.LatencyHistory))

			// Normalize latency factor (lower is better)
			latencyFactor := float64(aco.nodeTimeoutBase) / float64(avgLatency)
			if latencyFactor > 1.0 {
				latencyFactor = 1.0
			}

			weight *= latencyFactor
		}

		params.NodeWeights[nodeID] = weight
	}

	return params
}

// VectorClockUpdate updates a node's vector clock
func (aco *AdaptiveConsistencyOrchestrator) VectorClockUpdate(nodeID string, updates map[string]uint64) {
	aco.mutex.Lock()
	defer aco.mutex.Unlock()

	node, exists := aco.nodeTelemetry[nodeID]
	if !exists {
		return
	}

	// Update vector clock entries
	for id, version := range updates {
		if current, ok := node.VectorClock[id]; !ok || version > current {
			node.VectorClock[id] = version
		}
	}
}

// ReportNetworkPartition manually reports a network partition event
func (aco *AdaptiveConsistencyOrchestrator) ReportNetworkPartition(affectedNodes []string) {
	aco.mutex.Lock()
	defer aco.mutex.Unlock()

	// Record the partition event
	aco.networkStats.PartitionEvents = append(aco.networkStats.PartitionEvents, time.Now())

	// Update availability for affected nodes
	for _, nodeID := range affectedNodes {
		aco.networkStats.NodeAvailability[nodeID] = 0.0
	}

	// Force consistency level to eventual consistency
	aco.currentConsistency = EventualConsistency
}

// GetNetworkStats returns the current network statistics
func (aco *AdaptiveConsistencyOrchestrator) GetNetworkStats() NetworkStats {
	aco.mutex.RLock()
	defer aco.mutex.RUnlock()

	// Return a copy to prevent concurrent modification
	statsCopy := NetworkStats{
		LastSyncTime:     aco.networkStats.LastSyncTime,
		NodeAvailability: make(map[string]float64),
		PartitionEvents:  make([]time.Time, len(aco.networkStats.PartitionEvents)),
	}

	// Copy slices
	statsCopy.LatencyHistory = append([]time.Duration{}, aco.networkStats.LatencyHistory...)
	statsCopy.FailureHistory = append([]bool{}, aco.networkStats.FailureHistory...)
	statsCopy.ResponseTimeWindow = append([]time.Duration{}, aco.networkStats.ResponseTimeWindow...)
	copy(statsCopy.PartitionEvents, aco.networkStats.PartitionEvents)

	// Copy map
	for k, v := range aco.networkStats.NodeAvailability {
		statsCopy.NodeAvailability[k] = v
	}

	return statsCopy
}
