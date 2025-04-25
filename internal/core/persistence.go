package core

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BlockchainPersistence provides storage capabilities for the blockchain
type BlockchainPersistence struct {
	dataDir          string
	mutex            sync.RWMutex
	blocksPerFile    int
	currentFileIndex int
	blocksInMemCache map[string]*Block  // Cache by hash
	heightCache      map[int64]*Block   // Cache by height
	fileMap          map[int]*BlockFile // Map file indices to block files
}

// BlockFile represents a file containing multiple blocks
type BlockFile struct {
	StartHeight int64
	EndHeight   int64
	Blocks      []*Block
	Dirty       bool
}

// NewBlockchainPersistence creates a new persistence manager
func NewBlockchainPersistence(dataDir string) (*BlockchainPersistence, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	bp := &BlockchainPersistence{
		dataDir:          dataDir,
		blocksPerFile:    100, // Store 100 blocks per file for efficient I/O
		blocksInMemCache: make(map[string]*Block),
		heightCache:      make(map[int64]*Block),
		fileMap:          make(map[int]*BlockFile),
	}

	// Initialize and load existing data
	if err := bp.initialize(); err != nil {
		return nil, err
	}

	return bp, nil
}

// initialize loads existing blockchain data from disk
func (bp *BlockchainPersistence) initialize() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	// Read index file if it exists
	indexPath := filepath.Join(bp.dataDir, "blockchain_index.json")
	if _, err := os.Stat(indexPath); err == nil {
		// Index exists, load it
		indexData, err := os.ReadFile(indexPath)
		if err != nil {
			return fmt.Errorf("failed to read blockchain index: %w", err)
		}

		var index struct {
			CurrentFileIndex int   `json:"current_file_index"`
			FileIndices      []int `json:"file_indices"`
		}

		if err := json.Unmarshal(indexData, &index); err != nil {
			return fmt.Errorf("failed to parse blockchain index: %w", err)
		}

		bp.currentFileIndex = index.CurrentFileIndex

		// Load block files listed in the index
		for _, fileIndex := range index.FileIndices {
			filePath := filepath.Join(bp.dataDir, fmt.Sprintf("blocks_%d.dat", fileIndex))
			blockFile, err := bp.loadBlockFile(filePath)
			if err != nil {
				return err
			}

			bp.fileMap[fileIndex] = blockFile

			// Add blocks to cache
			for _, block := range blockFile.Blocks {
				bp.blocksInMemCache[string(block.Hash)] = block
				bp.heightCache[block.Height] = block
			}
		}
	}

	return nil
}

// loadBlockFile loads blocks from a block file
func (bp *BlockchainPersistence) loadBlockFile(filePath string) (*BlockFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read block file %s: %w", filePath, err)
	}

	blockFile := &BlockFile{
		Blocks: make([]*Block, 0),
		Dirty:  false,
	}

	// Deserialize blocks
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	var blockCount int
	if err := decoder.Decode(&blockCount); err != nil {
		return nil, fmt.Errorf("failed to decode block count: %w", err)
	}

	for i := 0; i < blockCount; i++ {
		var block Block
		if err := decoder.Decode(&block); err != nil {
			return nil, fmt.Errorf("failed to decode block %d: %w", i, err)
		}
		blockFile.Blocks = append(blockFile.Blocks, &block)
	}

	// Set height range
	if len(blockFile.Blocks) > 0 {
		blockFile.StartHeight = blockFile.Blocks[0].Height
		blockFile.EndHeight = blockFile.Blocks[len(blockFile.Blocks)-1].Height
	}

	return blockFile, nil
}

// SaveBlock persists a block to storage
func (bp *BlockchainPersistence) SaveBlock(block *Block) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	// Add to in-memory cache
	bp.blocksInMemCache[string(block.Hash)] = block
	bp.heightCache[block.Height] = block

	// Determine which file this block belongs to
	fileIndex := int(block.Height / int64(bp.blocksPerFile))

	// Get or create the file
	blockFile, exists := bp.fileMap[fileIndex]
	if !exists {
		blockFile = &BlockFile{
			Blocks:      make([]*Block, 0, bp.blocksPerFile),
			Dirty:       true,
			StartHeight: block.Height,
			EndHeight:   block.Height,
		}
		bp.fileMap[fileIndex] = blockFile
	} else {
		blockFile.Dirty = true
		if block.Height < blockFile.StartHeight {
			blockFile.StartHeight = block.Height
		}
		if block.Height > blockFile.EndHeight {
			blockFile.EndHeight = block.Height
		}
	}

	// Add block to file
	blockFile.Blocks = append(blockFile.Blocks, block)

	// Update current file index if needed
	if fileIndex > bp.currentFileIndex {
		bp.currentFileIndex = fileIndex
	}

	// Save the block file immediately for durability
	return bp.saveBlockFile(fileIndex)
}

// saveBlockFile writes a block file to disk
func (bp *BlockchainPersistence) saveBlockFile(fileIndex int) error {
	blockFile, exists := bp.fileMap[fileIndex]
	if !exists || !blockFile.Dirty {
		return nil
	}

	// Sort blocks by height (just in case)
	// This would be more efficient with sort.Slice and a custom Less function
	// but for simplicity we'll just use a bubble sort here
	for i := 0; i < len(blockFile.Blocks)-1; i++ {
		for j := 0; j < len(blockFile.Blocks)-i-1; j++ {
			if blockFile.Blocks[j].Height > blockFile.Blocks[j+1].Height {
				blockFile.Blocks[j], blockFile.Blocks[j+1] = blockFile.Blocks[j+1], blockFile.Blocks[j]
			}
		}
	}

	// Serialize blocks
	buf := bytes.Buffer{}
	encoder := gob.NewEncoder(&buf)

	// Write block count
	if err := encoder.Encode(len(blockFile.Blocks)); err != nil {
		return fmt.Errorf("failed to encode block count: %w", err)
	}

	// Write blocks
	for _, block := range blockFile.Blocks {
		if err := encoder.Encode(block); err != nil {
			return fmt.Errorf("failed to encode block: %w", err)
		}
	}

	// Write to file
	filePath := filepath.Join(bp.dataDir, fmt.Sprintf("blocks_%d.dat", fileIndex))
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write block file: %w", err)
	}

	blockFile.Dirty = false

	// Update index
	return bp.saveIndex()
}

// saveIndex writes the blockchain index to disk
func (bp *BlockchainPersistence) saveIndex() error {
	fileIndices := make([]int, 0, len(bp.fileMap))
	for idx := range bp.fileMap {
		fileIndices = append(fileIndices, idx)
	}

	index := struct {
		CurrentFileIndex int   `json:"current_file_index"`
		FileIndices      []int `json:"file_indices"`
	}{
		CurrentFileIndex: bp.currentFileIndex,
		FileIndices:      fileIndices,
	}

	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize blockchain index: %w", err)
	}

	indexPath := filepath.Join(bp.dataDir, "blockchain_index.json")
	if err := os.WriteFile(indexPath, indexData, 0644); err != nil {
		return fmt.Errorf("failed to write blockchain index: %w", err)
	}

	return nil
}

// GetBlockByHash retrieves a block by its hash
func (bp *BlockchainPersistence) GetBlockByHash(hash []byte) (*Block, error) {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	// Check in-memory cache first
	if block, exists := bp.blocksInMemCache[string(hash)]; exists {
		return block, nil
	}

	// Not in memory, need to load from disk
	// This would require a hash -> file index mapping for efficiency
	// For now, we'll just search all files
	for _, blockFile := range bp.fileMap {
		for _, block := range blockFile.Blocks {
			if bytes.Equal(block.Hash, hash) {
				// Add to cache for future lookups
				bp.blocksInMemCache[string(hash)] = block
				return block, nil
			}
		}
	}

	return nil, errors.New("block not found")
}

// GetBlockByHeight retrieves a block by its height
func (bp *BlockchainPersistence) GetBlockByHeight(height int64) (*Block, error) {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	// Check in-memory cache first
	if block, exists := bp.heightCache[height]; exists {
		return block, nil
	}

	// Determine which file would contain this height
	fileIndex := int(height / int64(bp.blocksPerFile))
	blockFile, exists := bp.fileMap[fileIndex]
	if !exists {
		return nil, errors.New("block height out of range")
	}

	// Check if the height is within this file's range
	if height < blockFile.StartHeight || height > blockFile.EndHeight {
		return nil, errors.New("block height out of range")
	}

	// Search for the block
	for _, block := range blockFile.Blocks {
		if block.Height == height {
			// Add to cache for future lookups
			bp.heightCache[height] = block
			return block, nil
		}
	}

	return nil, errors.New("block not found")
}

// GetLatestBlock retrieves the most recent block
func (bp *BlockchainPersistence) GetLatestBlock() (*Block, error) {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	// Find the highest height in our cache
	var maxHeight int64 = -1
	var latestBlock *Block

	// This could be optimized with a separate tracking variable
	for _, block := range bp.heightCache {
		if block.Height > maxHeight {
			maxHeight = block.Height
			latestBlock = block
		}
	}

	if latestBlock != nil {
		return latestBlock, nil
	}

	return nil, errors.New("no blocks found")
}

// Close flushes all pending writes and closes the persistence manager
func (bp *BlockchainPersistence) Close() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	// Save any dirty files
	for fileIndex, blockFile := range bp.fileMap {
		if blockFile.Dirty {
			if err := bp.saveBlockFile(fileIndex); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetBlockHeight returns the current blockchain height
func (bp *BlockchainPersistence) GetBlockHeight() (int64, error) {
	latestBlock, err := bp.GetLatestBlock()
	if err != nil {
		return -1, err
	}
	return latestBlock.Height, nil
}

// GetBlocksBetweenHeights returns blocks within a height range
func (bp *BlockchainPersistence) GetBlocksBetweenHeights(startHeight, endHeight int64) ([]*Block, error) {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	if startHeight > endHeight {
		return nil, errors.New("invalid height range")
	}

	result := make([]*Block, 0)

	// For each height in the range, try to get the block
	for h := startHeight; h <= endHeight; h++ {
		if block, exists := bp.heightCache[h]; exists {
			result = append(result, block)
			continue
		}

		// Not in cache, check file
		fileIndex := int(h / int64(bp.blocksPerFile))
		blockFile, exists := bp.fileMap[fileIndex]
		if !exists {
			continue // Skip this height
		}

		for _, block := range blockFile.Blocks {
			if block.Height == h {
				result = append(result, block)
				// Add to cache
				bp.heightCache[h] = block
				break
			}
		}
	}

	return result, nil
}
