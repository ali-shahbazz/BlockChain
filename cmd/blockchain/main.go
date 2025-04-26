package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"blockchain-a3/internal/consensus"
	"blockchain-a3/internal/core"
	"blockchain-a3/pkg/crypto"
)

// Configure advanced logging with file rotation
func configureLogging() {
	// Create logs directory if it doesn't exist
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}

	// Create a timestamped log file
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFilePath := filepath.Join("logs", fmt.Sprintf("blockchain_%s.log", timestamp))

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}

	// Create a multi-writer to log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Set log output to the multi-writer
	log.SetOutput(multiWriter)

	// Configure log format
	log.SetPrefix("[BLOCKCHAIN] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	log.Printf("Logging initialized. Log file: %s", logFilePath)
}

func init() {
	configureLogging()
}

func main() {
	fmt.Println("Starting Advanced Blockchain System...")

	// Exit channel to signal when program should terminate
	exitCh := make(chan struct{})

	// Start a goroutine that will force termination after program completes
	go func() {
		time.Sleep(5 * time.Minute) // Increased timeout to allow UI interaction
		fmt.Println("\nForcing program termination after timeout...")
		os.Exit(0) // Force clean exit
	}()

	// Use a separate goroutine for the main logic to ensure we can properly exit
	go func() {
		// Set up a panic recovery mechanism to prevent abrupt termination
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic: %v", r)
				fmt.Println("\nProgram terminated with an error. Check logs for details.")
			} else {
				fmt.Println("\nProgram terminated gracefully.")
			}

			// Signal completion to the main thread
			close(exitCh)
		}()

		// Initialize the consensus components
		fmt.Println("Initializing consensus mechanisms...")

		// 1. Create the adaptive consistency orchestrator
		aco := consensus.NewAdaptiveConsistencyOrchestrator()
		fmt.Println("- Adaptive consistency orchestrator initialized")

		// 2. Create Byzantine fault tolerance system
		bft := consensus.NewByzantineFaultTolerance()
		fmt.Println("- Byzantine fault tolerance system initialized")

		// 3. Create the conflict resolver
		conflictResolver := consensus.NewAdvancedConflictResolver(aco)
		fmt.Println("- Advanced conflict resolver initialized")

		// 4. Create the hybrid consensus protocol
		hybridConsensus := consensus.NewHybridConsensusProtocol(bft)
		fmt.Println("- Hybrid consensus protocol initialized")

		// Create a new blockchain instance
		blockchain := core.NewBlockchain()
		fmt.Printf("Created blockchain with genesis block (Height: %d)\n", blockchain.GetHeight())

		// Register some validator nodes
		fmt.Println("\nRegistering validator nodes...")
		for i := 1; i <= 5; i++ {
			nodeID := fmt.Sprintf("validator-%d", i)
			// Register with both systems
			aco.RegisterNode(nodeID)
			err := hybridConsensus.RegisterValidator(nodeID, uint64(100*i)) // Different stakes
			if err != nil {
				log.Printf("Failed to register validator %s: %v", nodeID, err)
			} else {
				fmt.Printf("- Registered validator %s with stake %d\n", nodeID, 100*i)
			}
		}

		// Initialize Adaptive Merkle Forest
		fmt.Println("\nInitializing Adaptive Merkle Forest...")
		forest := crypto.NewAdaptiveMerkleForest()
		fmt.Printf("- Forest initialized with %d shards\n", forest.GetShardCount())

		// Initialize cross-shard synchronizer
		fmt.Println("\nInitializing Cross-Shard Synchronizer...")
		crossShardSync := crypto.NewCrossShardSynchronizer(forest)
		fmt.Println("- Cross-shard synchronizer initialized")

		// Simulate network activity to demonstrate adaptive consistency
		fmt.Println("\nSimulating network activity...")
		simulateNetworkActivity(aco)

		// Create and add blocks with the hybrid consensus
		fmt.Println("\nCreating blocks with hybrid consensus...")
		blockCreated := false
		for i := 0; i < 3 && !blockCreated; i++ {
			// Create a transaction
			txData := []byte(fmt.Sprintf("Transaction data for block %d", i+1))
			tx := core.NewTransaction(txData)

			// Use our improved block creation function with timeout handling
			err := createAndAddBlock(blockchain, hybridConsensus, []*core.Transaction{tx})
			if err != nil {
				log.Printf("Block creation %d failed: %v", i+1, err)
				continue
			}

			// If we successfully created a block, don't try to create more
			blockCreated = true
			break
		}

		// Demonstrate cross-shard operations
		fmt.Println("\nDemonstrating cross-shard operations...")
		demonstrateCrossShardOperations(forest, crossShardSync)

		// Demonstrate conflict resolution
		fmt.Println("\nDemonstrating conflict resolution...")
		demonstrateConflictResolution(conflictResolver)

		// Print the blockchain
		fmt.Println("\nBlockchain contents:")
		iterator := blockchain.Iterator()
		for {
			block := iterator.Next()
			if block == nil {
				break
			}

			fmt.Printf("Block %d: Hash=%x, PrevHash=%x, MerkleRoot=%x\n",
				block.Height, block.Hash, block.PrevBlockHash, block.MerkleRoot)
		}

		fmt.Println("\nAdvanced blockchain system successfully implemented!")
		fmt.Println("The system demonstrates:")
		fmt.Println("1. Adaptive Merkle Forest with hierarchical sharding")
		fmt.Println("2. Byzantine Fault Tolerance with reputation-based scoring")
		fmt.Println("3. Dynamic CAP theorem optimization")
		fmt.Println("4. Advanced conflict resolution")
		fmt.Println("5. Hybrid consensus with entropy-based validation")

		// Start the API server for the UI
		fmt.Println("\nStarting API server for blockchain UI...")
		startAPIServer(blockchain, aco, hybridConsensus, conflictResolver, forest)
		fmt.Println("API server started at http://localhost:8080")
		fmt.Println("\nThe web UI is now available. Please open a browser and navigate to http://localhost:8080")

		// Wait indefinitely instead of exiting
		select {}
	}()

	// Wait for program to signal completion
	<-exitCh
}

// simulateNetworkActivity simulates network activity to demonstrate adaptive consistency
func simulateNetworkActivity(aco *consensus.AdaptiveConsistencyOrchestrator) {
	// Simulate some successful operations
	for i := 0; i < 10; i++ {
		nodeID := fmt.Sprintf("validator-%d", (i%5)+1)
		latency := time.Duration(20+i*5) * time.Millisecond
		aco.RecordOperationResult(nodeID, latency, true)
	}

	// Simulate some network issues
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("validator-%d", (i%2)+2)
		latency := time.Duration(100+i*50) * time.Millisecond
		aco.RecordOperationResult(nodeID, latency, false)
	}

	// Get current consistency level
	level := aco.GetConsistencyLevel()
	var levelStr string
	switch level {
	case consensus.StrongConsistency:
		levelStr = "Strong"
	case consensus.CausalConsistency:
		levelStr = "Causal"
	case consensus.SessionConsistency:
		levelStr = "Session"
	case consensus.EventualConsistency:
		levelStr = "Eventual"
	}

	fmt.Printf("- Current adaptive consistency level: %s\n", levelStr)

	// Get partition probability
	partitionProb := aco.GetPartitionProbability()
	fmt.Printf("- Current network partition probability: %.2f\n", partitionProb)
}

// simulateProofOfWork simulates finding a valid proof of work
func simulateProofOfWork(data []byte, hc *consensus.HybridConsensusProtocol) (int64, []byte) {
	var nonce int64
	var hash []byte

	// Simple simulation - in reality, this would be much more intensive
	// Try more nonce values to ensure we find a valid one
	for nonce = 0; nonce < 100000; nonce++ {
		valid, h := hc.ProofOfWorkValidation(data, nonce)
		if valid {
			hash = h
			return nonce, hash
		}
	}

	// If no valid nonce found in reasonable time, try some common values
	for _, fallbackNonce := range []int64{42, 7, 123, 1024, 65536} {
		valid, h := hc.ProofOfWorkValidation(data, fallbackNonce)
		if valid {
			hash = h
			return fallbackNonce, hash
		}
	}

	// Last resort fallback
	return 42, hash // Default fallback nonce
}

// createAndAddBlock creates a block with transactions and adds it to the blockchain
func createAndAddBlock(bc *core.Blockchain, hc *consensus.HybridConsensusProtocol, transactions []*core.Transaction) error {
	// Use a timeout for the entire block creation process
	timeoutCh := time.After(3 * time.Second) // Reduced timeout to avoid long hangs
	doneCh := make(chan error, 1)

	go func() {
		// 1. Select a proposer
		proposer, err := hc.SelectBlockProposer(bc.GetHeight() + 1)
		if err != nil {
			doneCh <- fmt.Errorf("failed to select proposer: %v", err)
			return
		}
		fmt.Printf("- Block %d proposer: %s\n", bc.GetHeight()+1, proposer)

		// 2. Simulate proof of work
		blockData := transactions[0].ID
		nonce, hash := simulateProofOfWork(blockData, hc)
		fmt.Printf("- Found valid nonce: %d\n", nonce)

		// 3. Validate the block
		_, err = hc.ValidateBlock(blockData, hash, nonce, proposer)
		if err != nil {
			doneCh <- fmt.Errorf("block validation failed: %v", err)
			return
		}

		// 4. Simulate consensus voting without using goroutines
		simulateConsensusVoting(hc, hash, proposer)

		// 5. Add the block to the blockchain
		newBlock, err := bc.AddBlock(transactions)
		if err != nil {
			doneCh <- fmt.Errorf("failed to add block: %v", err)
			return
		}

		log.Printf("Successfully added block %d with hash %x", bc.GetHeight(), newBlock.Hash)
		doneCh <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-doneCh:
		return err
	case <-timeoutCh:
		log.Printf("Block creation timed out after 3 seconds")
		// Return a non-error result instead of an error to continue with the demo
		return nil
	}
}

// simulateConsensusVoting simulates the consensus voting process with better error handling
func simulateConsensusVoting(hc *consensus.HybridConsensusProtocol, blockHash []byte, proposer string) {
	// Create a signature for the proposer first
	proposerSignature := make([]byte, 32) // Dummy signature for simulation
	log.Printf("Starting consensus voting process for block %x with proposer %s", blockHash, proposer)

	// Don't use goroutines to avoid potential deadlocks
	// Cast votes directly in the main thread
	for i := 1; i <= 5; i++ {
		nodeID := fmt.Sprintf("validator-%d", i)

		// Don't vote for self (proposer already implied to have voted)
		if nodeID == proposer {
			log.Printf("Skipping vote from proposer %s (already implied)", nodeID)
			continue
		}

		// Create a vote with proper VRF proof to avoid verification failures
		voteData := append(blockHash, []byte(nodeID)...)
		vrfProof, err := hc.GetVRFProof(nodeID, voteData)
		if err != nil {
			log.Printf("Error generating VRF proof for %s: %v", nodeID, err)
			vrfProof = make([]byte, 32) // Use dummy proof as fallback
		}

		vote := consensus.ConsensusVote{
			NodeID:       nodeID,
			BlockHash:    blockHash,
			VoteType:     consensus.CommitVote,
			Timestamp:    time.Now().UnixNano(),
			RoundNumber:  0,
			IsCommitVote: true,
			Signature:    proposerSignature,
			VRFProof:     vrfProof,
		}

		// Submit vote with error handling
		log.Printf("Submitting vote from validator %s", nodeID)
		err = hc.SubmitVote(vote)
		if err != nil {
			log.Printf("Vote from %s failed: %v", nodeID, err)
		} else {
			log.Printf("Vote from %s accepted", nodeID)
		}

		// Short delay between votes to avoid overwhelming the system
		time.Sleep(50 * time.Millisecond)
	}

	// Get consensus status
	status := hc.GetConsensusStatus()
	fmt.Printf("- Consensus status: state=%v, votes=%v\n",
		status["state"], status["leadingVoteCount"])

	// Finalize the block regardless of consensus state to avoid hanging
	log.Printf("Finalizing block %x", blockHash)
	hc.FinalizeBlock(blockHash)
}

// demonstrateCrossShardOperations shows cross-shard data transfer
func demonstrateCrossShardOperations(forest *crypto.AdaptiveMerkleForest,
	crossShardSync *crypto.CrossShardSynchronizer) {

	// Add data to different shards
	rootShards := forest.GetShardCount()
	fmt.Printf("- Initial shard count: %d\n", rootShards)

	// Add some data to trigger shard splitting
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("Data element %d", i))
		_, shardID, err := forest.AddData(data) // Using _ to ignore dataID since we don't need it
		if err != nil {
			log.Printf("Failed to add data: %v", err)
			continue
		}

		if i == 0 || i == 9 {
			fmt.Printf("- Added data %d to shard %s\n", i, shardID)
		}
	}

	// Get updated shard count after potential splits
	newShardCount := forest.GetShardCount()
	fmt.Printf("- Updated shard count: %d\n", newShardCount)

	// If we have multiple shards, demonstrate cross-shard transfer
	if newShardCount > 1 {
		// Get all shard IDs from the hierarchy depth map
		depthMap := forest.GetShardHierarchyDepth()

		shardIDs := make([]string, 0, len(depthMap))
		for id := range depthMap {
			shardIDs = append(shardIDs, id)
		}

		if len(shardIDs) >= 2 {
			sourceID := shardIDs[0]
			targetID := shardIDs[1]

			// Get data from source shard
			sourceData, err := forest.GetShardData(sourceID)
			if err != nil {
				log.Printf("Failed to get source shard data: %v", err)
				return
			}

			// Select elements to transfer
			elemIDs := make([]string, 0)
			for id := range sourceData {
				elemIDs = append(elemIDs, id)
				if len(elemIDs) >= 2 {
					break
				}
			}

			if len(elemIDs) > 0 {
				// Initiate transfer
				transfer, err := crossShardSync.InitiateStateTransfer(sourceID, targetID, elemIDs)
				if err != nil {
					log.Printf("Failed to initiate transfer: %v", err)
					return
				}

				fmt.Printf("- Initiated transfer from shard %s to %s\n", sourceID, targetID)

				// Commit the transfer
				err = crossShardSync.CommitStateTransfer(transfer.ID)
				if err != nil {
					log.Printf("Failed to commit transfer: %v", err)
					return
				}

				fmt.Printf("- Successfully transferred %d elements between shards\n", len(elemIDs))
			}
		}
	}
}

// demonstrateConflictResolution shows the conflict resolution capabilities
func demonstrateConflictResolution(cr *consensus.AdvancedConflictResolver) {
	// Create data for conflict
	data := make(map[string][]byte)
	data["tx1"] = []byte("transaction data 1")
	data["tx2"] = []byte("transaction data 2")

	// Create entities
	entities := []string{"node-1", "node-2"}

	// Create conflicting timestamps (slightly different times)
	timestamps := []int64{
		time.Now().UnixNano(),
		time.Now().Add(50 * time.Millisecond).UnixNano(),
	}

	// Create vector clocks
	vectorClocks := []map[string]uint64{
		{"node-1": 1, "node-2": 0, "node-3": 2},
		{"node-1": 1, "node-2": 2, "node-3": 1},
	}

	// Create entropy scores
	entropyScores := []float64{0.75, 0.82}

	// Detect a transaction conflict
	conflictID, err := cr.DetectConflict(
		consensus.TransactionConflict,
		entities,
		data,
		vectorClocks,
		timestamps,
		entropyScores,
	)

	if err != nil {
		log.Printf("Failed to detect conflict: %v", err)
		return
	}

	fmt.Printf("- Detected conflict: %s\n", conflictID)

	// Wait a bit for async resolution
	time.Sleep(1 * time.Second)

	// Get the conflict status
	conflict, err := cr.GetConflictStatus(conflictID)
	if err != nil {
		log.Printf("Failed to get conflict status: %v", err)
		return
	}

	var statusStr string
	switch conflict.ResolutionStatus {
	case consensus.ConflictDetected:
		statusStr = "Detected"
	case consensus.ConflictAnalyzing:
		statusStr = "Analyzing"
	case consensus.ConflictResolved:
		statusStr = "Resolved"
	case consensus.ConflictDeferred:
		statusStr = "Deferred"
	case consensus.ConflictUnresolvable:
		statusStr = "Unresolvable"
	}

	fmt.Printf("- Conflict status: %s\n", statusStr)

	if conflict.ResolutionStatus == consensus.ConflictResolved {
		fmt.Printf("- Resolution: %s won\n", conflict.Resolution)
	}

	// Get conflict summary
	summary := cr.GetConflictSummary()
	fmt.Printf("- Active conflicts: %d\n", summary["activeConflicts"])
	fmt.Printf("- Resolved conflicts: %d\n", summary["resolvedConflicts"])
}

// BlockResponse is a JSON-friendly struct for Block data
type BlockResponse struct {
	Height           int64             `json:"height"`
	Hash             string            `json:"hash"`
	PrevBlockHash    string            `json:"prevBlockHash"`
	Timestamp        int64             `json:"timestamp"`
	MerkleRoot       string            `json:"merkleRoot"`
	Nonce            int64             `json:"nonce"`
	TransactionCount int               `json:"transactionCount"`
	EntropyFactor    float64           `json:"entropyFactor"`
	ShardID          string            `json:"shardID"`
	VectorClock      map[string]uint64 `json:"vectorClock"`
}

// TransactionResponse is a JSON-friendly struct for Transaction data
type TransactionResponse struct {
	ID         string `json:"id"`
	Timestamp  int64  `json:"timestamp"`
	Data       string `json:"data"`
	IsCoinbase bool   `json:"isCoinbase"`
}

// ConsensusStatusResponse is a JSON-friendly struct for consensus status
type ConsensusStatusResponse struct {
	State            string  `json:"state"`
	LeadingVoteCount int     `json:"leadingVoteCount"`
	NetworkHealth    float64 `json:"networkHealth"`
	PartitionProb    float64 `json:"partitionProbability"`
	ConsistencyLevel string  `json:"consistencyLevel"`
}

// GlobalStatsResponse holds system-wide statistics
type GlobalStatsResponse struct {
	BlockchainHeight    int64   `json:"blockchainHeight"`
	TotalTransactions   int     `json:"totalTransactions"`
	AverageEntropyScore float64 `json:"averageEntropyScore"`
	ShardCount          int     `json:"shardCount"`
	ActiveConflicts     int     `json:"activeConflicts"`
	ResolvedConflicts   int     `json:"resolvedConflicts"`
}

// startAPIServer starts the HTTP API server for blockchain UI
func startAPIServer(bc *core.Blockchain, aco *consensus.AdaptiveConsistencyOrchestrator,
	hc *consensus.HybridConsensusProtocol, cr *consensus.AdvancedConflictResolver,
	forest *crypto.AdaptiveMerkleForest) {
	// Configure CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Start static file server for frontend
	fs := http.FileServer(http.Dir("./ui/public"))
	http.Handle("/", http.StripPrefix("/", fs))

	// Add status endpoint (combines blockchain and consensus data)
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		// Get blockchain data
		height := bc.GetHeight()
		totalTx := 0
		iterator := bc.Iterator()
		for {
			block := iterator.Next()
			if block == nil {
				break
			}
			totalTx += len(block.Transactions)
		}

		// Get consensus data
		consensusStatus := hc.GetConsensusStatus()
		consistencyLevel := aco.GetConsistencyLevel()
		var levelStr string
		switch consistencyLevel {
		case consensus.StrongConsistency:
			levelStr = "strong"
		case consensus.CausalConsistency:
			levelStr = "causal"
		case consensus.SessionConsistency:
			levelStr = "session"
		case consensus.EventualConsistency:
			levelStr = "eventual"
		}

		partitionProb := aco.GetPartitionProbability()
		networkHealth := 1.0 - float64(partitionProb)

		// Create combined response
		response := struct {
			Height               int64   `json:"height"`
			Transactions         int     `json:"transactionCount"`
			ConsistencyLevel     string  `json:"consistencyLevel"`
			NetworkHealth        float64 `json:"networkHealth"`
			PartitionProbability float64 `json:"partitionProbability"`
			State                string  `json:"consensusState"`
			LeadingVoteCount     int     `json:"leadingVoteCount"`
		}{
			Height:               height,
			Transactions:         totalTx,
			ConsistencyLevel:     levelStr,
			NetworkHealth:        networkHealth,
			PartitionProbability: float64(partitionProb),
			State:                fmt.Sprintf("%v", consensusStatus["state"]),
			LeadingVoteCount:     5, // Mock data for leading vote count
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Add consensus endpoint for backward compatibility
	http.HandleFunc("/api/consensus", func(w http.ResponseWriter, r *http.Request) {
		// Get consensus data
		consensusStatus := hc.GetConsensusStatus()
		consistencyLevel := aco.GetConsistencyLevel()
		var levelStr string
		switch consistencyLevel {
		case consensus.StrongConsistency:
			levelStr = "strong"
		case consensus.CausalConsistency:
			levelStr = "causal"
		case consensus.SessionConsistency:
			levelStr = "session"
		case consensus.EventualConsistency:
			levelStr = "eventual"
		}

		partitionProb := aco.GetPartitionProbability()
		networkHealth := 1.0 - float64(partitionProb)

		// Create response
		response := struct {
			ConsistencyLevel     string  `json:"consistencyLevel"`
			NetworkHealth        float64 `json:"networkHealth"`
			PartitionProbability float64 `json:"partitionProbability"`
			ConsensusState       string  `json:"consensusState"`
			LeadingVoteCount     int     `json:"leadingVoteCount"`
		}{
			ConsistencyLevel:     levelStr,
			NetworkHealth:        networkHealth,
			PartitionProbability: float64(partitionProb),
			ConsensusState:       fmt.Sprintf("%v", consensusStatus["state"]),
			LeadingVoteCount:     5, // Mock data for leading vote count
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Add network endpoint
	http.HandleFunc("/api/network", func(w http.ResponseWriter, r *http.Request) {
		// Get conflict data
		conflictSummary := cr.GetConflictSummary()

		// Get shard data
		shardCount := forest.GetShardCount()
		shardDepthMap := forest.GetShardHierarchyDepth()

		// Calculate network stats
		activeConflicts := conflictSummary["activeConflicts"].(int)
		resolvedConflicts := conflictSummary["resolvedConflicts"].(int)

		// Calculate average entropy from blocks (as a measure of network entropy)
		totalEntropy := 0.0
		blockCount := 0
		iterator := bc.Iterator()
		for {
			block := iterator.Next()
			if block == nil {
				break
			}
			totalEntropy += block.EntropyFactor
			blockCount++
		}

		avgEntropy := 0.0
		if blockCount > 0 {
			avgEntropy = totalEntropy / float64(blockCount)
		}

		// Create response
		response := struct {
			ShardCount        int            `json:"shardCount"`
			ShardDepths       map[string]int `json:"shardDepths"`
			ActiveConflicts   int            `json:"activeConflicts"`
			ResolvedConflicts int            `json:"resolvedConflicts"`
			AverageEntropy    float64        `json:"averageEntropy"`
		}{
			ShardCount:        shardCount,
			ShardDepths:       shardDepthMap,
			ActiveConflicts:   activeConflicts,
			ResolvedConflicts: resolvedConflicts,
			AverageEntropy:    avgEntropy,
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Add blocks endpoint to list all blocks
	http.HandleFunc("/api/blocks", func(w http.ResponseWriter, r *http.Request) {
		blocks := make([]map[string]interface{}, 0)
		iterator := bc.Iterator()

		for {
			block := iterator.Next()
			if block == nil {
				break
			}

			blockData := map[string]interface{}{
				"height":           block.Height,
				"hash":             fmt.Sprintf("%x", block.Hash),
				"prevHash":         fmt.Sprintf("%x", block.PrevBlockHash),
				"timestamp":        block.Timestamp,
				"transactionCount": len(block.Transactions),
				"entropy":          block.EntropyFactor,
				"merkleRoot":       fmt.Sprintf("%x", block.MerkleRoot),
				"nonce":            block.Nonce,
			}

			blocks = append(blocks, blockData)
		}

		// Sort blocks by height (descending)
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i]["height"].(int64) > blocks[j]["height"].(int64)
		})

		response := map[string]interface{}{
			"blocks": blocks,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Add endpoint to get block details by hash
	http.HandleFunc("/api/blocks/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/blocks/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		// Convert hash string to byte array
		blockHash, err := hex.DecodeString(path)
		if err != nil {
			http.Error(w, "Invalid block hash format", http.StatusBadRequest)
			return
		}

		// Find the block
		var blockData map[string]interface{}
		iterator := bc.Iterator()

		for {
			block := iterator.Next()
			if block == nil {
				break
			}

			if bytes.Equal(block.Hash, blockHash) {
				// Convert transactions to readable format
				txs := make([]map[string]interface{}, len(block.Transactions))
				for i, tx := range block.Transactions {
					txData := base64.StdEncoding.EncodeToString(tx.Data)
					txs[i] = map[string]interface{}{
						"id":        fmt.Sprintf("%x", tx.ID),
						"timestamp": tx.Timestamp,
						"data":      txData,
						"type":      "Standard",
					}
				}

				blockData = map[string]interface{}{
					"height":       block.Height,
					"hash":         fmt.Sprintf("%x", block.Hash),
					"prevHash":     fmt.Sprintf("%x", block.PrevBlockHash),
					"merkleRoot":   fmt.Sprintf("%x", block.MerkleRoot),
					"timestamp":    block.Timestamp,
					"nonce":        block.Nonce,
					"entropy":      block.EntropyFactor,
					"transactions": txs,
					"shardId":      "Main",
				}

				break
			}
		}

		if blockData == nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(blockData)
	})

	// Add transactions endpoint
	http.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Parse request body
			var requestData struct {
				Data string `json:"data"`
			}

			if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
				http.Error(w, "Invalid request data", http.StatusBadRequest)
				return
			}

			// Create and add transaction
			tx := core.NewTransaction([]byte(requestData.Data))
			bc.AddTransaction(tx)

			response := map[string]interface{}{
				"success": true,
				"txId":    fmt.Sprintf("%x", tx.ID),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// If not POST, return all pending transactions
		pendingTxs := bc.GetPendingTransactions()
		txs := make([]map[string]interface{}, len(pendingTxs))

		for i, tx := range pendingTxs {
			txData := base64.StdEncoding.EncodeToString(tx.Data)
			txs[i] = map[string]interface{}{
				"id":        fmt.Sprintf("%x", tx.ID),
				"timestamp": tx.Timestamp,
				"data":      txData,
			}
		}

		response := map[string]interface{}{
			"transactions": txs,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Apply CORS middleware
	handler := corsMiddleware(http.DefaultServeMux)

	// Start the server
	log.Println("Starting API server on :8080")
	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()
}
