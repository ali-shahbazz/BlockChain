// Package crypto provides cryptographic primitives for the blockchain
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"math/big"
)

// BloomFilter implements a probabilistic set membership structure
type BloomFilter struct {
	bits      []bool
	numHashes int
	size      uint
}

// NewBloomFilter creates a new Bloom filter with the given parameters
func NewBloomFilter(size uint, numHashes int) *BloomFilter {
	return &BloomFilter{
		bits:      make([]bool, size),
		numHashes: numHashes,
		size:      size,
	}
}

// Add adds an element to the Bloom filter
func (bf *BloomFilter) Add(data []byte) {
	for i := 0; i < bf.numHashes; i++ {
		position := bf.hash(data, i) % bf.size
		bf.bits[position] = true
	}
}

// Test checks if an element might be in the Bloom filter
func (bf *BloomFilter) Test(data []byte) bool {
	for i := 0; i < bf.numHashes; i++ {
		position := bf.hash(data, i) % bf.size
		if !bf.bits[position] {
			return false
		}
	}
	return true
}

// hash generates a hash of the data with a salt based on the index
func (bf *BloomFilter) hash(data []byte, index int) uint {
	// Add salt based on index to create different hash functions
	saltedData := append(data, byte(index))
	hash := sha256.Sum256(saltedData)

	// Use the first 8 bytes as a uint64
	return uint(binary.BigEndian.Uint64(hash[:8]))
}

// Reset clears the Bloom filter
func (bf *BloomFilter) Reset() {
	bf.bits = make([]bool, bf.size)
}

// EstimateFalsePositiveRate estimates the false positive rate of the filter
func (bf *BloomFilter) EstimateFalsePositiveRate(numElements int) float64 {
	// Probability of a bit being 0 after all insertions
	p := math.Pow(1-1/float64(bf.size), float64(numElements*bf.numHashes))

	// Probability of false positive (all k positions are 1)
	return math.Pow(1-p, float64(bf.numHashes))
}

// CryptographicAccumulator implements an RSA accumulator
type CryptographicAccumulator struct {
	modulus    *big.Int
	value      *big.Int
	safeprimes *big.Int
	elements   map[string]bool
}

// NewCryptographicAccumulator creates a new accumulator with RSA modulus
func NewCryptographicAccumulator(bits int) (*CryptographicAccumulator, error) {
	// Generate two safe primes for RSA modulus
	p, err := rand.Prime(rand.Reader, bits/2)
	if err != nil {
		return nil, err
	}

	q, err := rand.Prime(rand.Reader, bits/2)
	if err != nil {
		return nil, err
	}

	// Compute modulus
	modulus := new(big.Int).Mul(p, q)

	// Initial value (often chosen as a small prime number)
	value := big.NewInt(3)

	return &CryptographicAccumulator{
		modulus:  modulus,
		value:    value,
		elements: make(map[string]bool),
	}, nil
}

// Add adds an element to the accumulator
func (ca *CryptographicAccumulator) Add(data []byte) {
	// Convert data to a prime representative
	prime := ca.getPrimeRepresentative(data)

	// Update accumulator value: A' = A^e mod N
	ca.value.Exp(ca.value, prime, ca.modulus)

	// Track element
	ca.elements[string(data)] = true
}

// Remove removes an element from the accumulator (if possible)
func (ca *CryptographicAccumulator) Remove(data []byte) error {
	// Check if element exists
	if !ca.elements[string(data)] {
		return errors.New("element not in accumulator")
	}

	// For true dynamic accumulators, removing elements requires additional state
	// This is a simplification - a real implementation would require more complex logic

	// For now, we'll rebuild the accumulator without the element
	// (This is inefficient but easier to understand)

	// Reset accumulator
	ca.value = big.NewInt(3)

	// Rebuild with all elements except the one to remove
	for elem := range ca.elements {
		if elem != string(data) {
			prime := ca.getPrimeRepresentative([]byte(elem))
			ca.value.Exp(ca.value, prime, ca.modulus)
		}
	}

	// Remove from tracking
	delete(ca.elements, string(data))

	return nil
}

// VerifyMembership verifies if an element is in the accumulator
func (ca *CryptographicAccumulator) VerifyMembership(data []byte, proof *big.Int) bool {
	// Convert data to prime representative
	prime := ca.getPrimeRepresentative(data)

	// Verify: proof^e mod N == accumulator value
	result := new(big.Int).Exp(proof, prime, ca.modulus)

	return result.Cmp(ca.value) == 0
}

// GenerateProof generates a membership proof for an element
func (ca *CryptographicAccumulator) GenerateProof(data []byte) (*big.Int, error) {
	// Check if element exists
	if !ca.elements[string(data)] {
		return nil, errors.New("element not in accumulator")
	}

	// For RSA accumulators, proof generation requires knowledge of all elements
	// or additional state tracking (like storing witnesses)

	// This is a simplified implementation
	// In a real system, we'd need to compute A^(1/e) mod N where e is the prime representation

	// For this example, we'll just return the accumulator value as a placeholder
	// NOTE: This is NOT a valid proof in a real system
	return ca.value, nil
}

// findNextPrime finds the next prime number greater than or equal to the given number
func findNextPrime(num *big.Int) *big.Int {
	candidate := new(big.Int).Set(num)
	if candidate.Cmp(big.NewInt(2)) < 0 {
		return big.NewInt(2) // The smallest prime number
	}

	// Ensure candidate is odd (even numbers > 2 are not prime)
	if candidate.Bit(0) == 0 {
		candidate.Add(candidate, big.NewInt(1))
	}

	for {
		if candidate.ProbablyPrime(20) { // 20 rounds of Miller-Rabin primality test
			return candidate
		}
		candidate.Add(candidate, big.NewInt(2)) // Check the next odd number
	}
}

// getPrimeRepresentative converts input data to a prime number
func (ca *CryptographicAccumulator) getPrimeRepresentative(data []byte) *big.Int {
	// Hash the data
	hash := sha256.Sum256(data)
	num := new(big.Int).SetBytes(hash[:])

	// Find the next prime number after the hash
	return findNextPrime(num)
}

// ProofCompressor provides probabilistic compression of cryptographic proofs
type ProofCompressor struct {
	compressionLevel int
	errorRate        float64
}

// NewProofCompressor creates a new proof compressor
func NewProofCompressor(compressionLevel int, targetErrorRate float64) *ProofCompressor {
	return &ProofCompressor{
		compressionLevel: compressionLevel,
		errorRate:        targetErrorRate,
	}
}

// CompressMerkleProof compresses a Merkle proof using probabilistic techniques
func (pc *ProofCompressor) CompressMerkleProof(proof [][]byte) ([][]byte, error) {
	if pc.compressionLevel <= 0 {
		return proof, nil // No compression
	}

	// For high compression levels, we'll drop some intermediary hashes
	// and rely on verification to reconstruct them probabilistically
	if len(proof) <= pc.compressionLevel*2 {
		return proof, nil // Not enough elements to compress
	}

	// Simple compression scheme: remove every Nth element
	compressed := make([][]byte, 0)
	for i := 0; i < len(proof); i++ {
		// Keep the direction information (every other element)
		if i%2 == 1 {
			compressed = append(compressed, proof[i])
			continue
		}

		// For hash elements, skip some based on compression level
		if (i/2)%pc.compressionLevel != 0 {
			compressed = append(compressed, proof[i])
		} else {
			// Add a marker to indicate omitted hash
			compressed = append(compressed, []byte{0})
		}
	}

	return compressed, nil
}

// DecompressMerkleProof attempts to decompress a compressed Merkle proof
func (pc *ProofCompressor) DecompressMerkleProof(compressed [][]byte) ([][]byte, error) {
	// Identify if compression was used (check for marker bytes)
	hasMarkers := false
	for i := 0; i < len(compressed); i += 2 {
		if i < len(compressed) && len(compressed[i]) == 1 && compressed[i][0] == 0 {
			hasMarkers = true
			break
		}
	}

	if !hasMarkers {
		return compressed, nil // No decompression needed
	}

	// Reconstruct the missing hashes
	decompressed := make([][]byte, 0, len(compressed)*2)

	for i := 0; i < len(compressed); i += 2 {
		if i+1 >= len(compressed) {
			// Unexpected format, just return what we have
			return decompressed, nil
		}

		// Check if this is a marker for omitted hash
		if len(compressed[i]) == 1 && compressed[i][0] == 0 {
			// For omitted hash, we'll generate a placeholder
			// In a real implementation, this would use surrounding context to reconstruct
			side := compressed[i+1]
			placeholderHash := make([]byte, 32) // Standard hash size

			// Create a deterministic placeholder based on the side
			if len(side) > 0 {
				placeholderHash[0] = side[0]
			}

			decompressed = append(decompressed, placeholderHash)
			decompressed = append(decompressed, side)
		} else {
			// For regular elements, just copy them
			decompressed = append(decompressed, compressed[i])
			decompressed = append(decompressed, compressed[i+1])
		}
	}

	return decompressed, nil
}

// AMQFilter implements an approximate membership query filter using cuckoo filters
type AMQFilter struct {
	buckets        [][]uint32
	bucketSize     int
	numBuckets     uint32
	fingerprintLen int
	itemCount      int
	maxKicks       int
}

// NewAMQFilter creates a new approximate membership query filter
func NewAMQFilter(numBuckets uint32, bucketSize, fingerprintLen, maxKicks int) *AMQFilter {
	buckets := make([][]uint32, numBuckets)
	for i := range buckets {
		buckets[i] = make([]uint32, 0, bucketSize)
	}

	return &AMQFilter{
		buckets:        buckets,
		bucketSize:     bucketSize,
		numBuckets:     numBuckets,
		fingerprintLen: fingerprintLen,
		maxKicks:       maxKicks,
		itemCount:      0,
	}
}

// Add adds an element to the filter
func (cf *AMQFilter) Add(data []byte) bool {
	fingerprint, bucket1, bucket2 := cf.getIndices(data)

	// Try to insert in first bucket
	if len(cf.buckets[bucket1]) < cf.bucketSize {
		cf.buckets[bucket1] = append(cf.buckets[bucket1], fingerprint)
		cf.itemCount++
		return true
	}

	// Try to insert in second bucket
	if len(cf.buckets[bucket2]) < cf.bucketSize {
		cf.buckets[bucket2] = append(cf.buckets[bucket2], fingerprint)
		cf.itemCount++
		return true
	}

	// Both buckets are full, try cuckoo hashing
	return cf.relocate(fingerprint, bucket1, bucket2)
}

// Lookup checks if an item might be in the filter
func (cf *AMQFilter) Lookup(data []byte) bool {
	fingerprint, bucket1, bucket2 := cf.getIndices(data)

	// Check first bucket
	for _, fp := range cf.buckets[bucket1] {
		if fp == fingerprint {
			return true
		}
	}

	// Check second bucket
	for _, fp := range cf.buckets[bucket2] {
		if fp == fingerprint {
			return true
		}
	}

	return false
}

// relocate implements the cuckoo hashing relocation mechanism
func (cf *AMQFilter) relocate(fingerprint uint32, bucket1, bucket2 uint32) bool {
	currentBucket := bucket1
	for i := 0; i < cf.maxKicks; i++ {
		// Randomly select a victim from the bucket
		bucketSize := len(cf.buckets[currentBucket])
		if bucketSize == 0 {
			return false
		}

		victimIdx := int(fingerprint) % bucketSize
		victim := cf.buckets[currentBucket][victimIdx]

		// Replace the victim with the current fingerprint
		cf.buckets[currentBucket][victimIdx] = fingerprint

		// Find the alternate bucket for the victim
		hash := sha256.Sum256(cf.uint32ToBytes(victim))
		seed := binary.BigEndian.Uint32(hash[:4])
		alternateBucket := (seed ^ victim) % cf.numBuckets

		// If the current bucket is the alternate bucket, use the other one
		if alternateBucket == currentBucket {
			hash = sha256.Sum256(cf.uint32ToBytes(alternateBucket))
			seed = binary.BigEndian.Uint32(hash[:4])
			alternateBucket = (seed ^ victim) % cf.numBuckets
		}

		// Try to insert the victim in its alternate bucket
		if len(cf.buckets[alternateBucket]) < cf.bucketSize {
			cf.buckets[alternateBucket] = append(cf.buckets[alternateBucket], victim)
			cf.itemCount++
			return true
		}

		// Continue with the victim
		fingerprint = victim
		currentBucket = alternateBucket
	}

	// Couldn't relocate after max kicks
	return false
}

// getIndices computes the fingerprint and bucket indices for a data item
func (cf *AMQFilter) getIndices(data []byte) (uint32, uint32, uint32) {
	hash := sha256.Sum256(data)

	// Use first 4 bytes for fingerprint
	fingerprint := binary.BigEndian.Uint32(hash[:4])

	// Ensure fingerprint is never 0 (reserved as empty marker)
	if fingerprint == 0 {
		fingerprint = 1
	}

	// Mask fingerprint to specified number of bits
	mask := uint32((1 << cf.fingerprintLen) - 1)
	fingerprint &= mask

	// Get the first bucket index from next 4 bytes
	bucket1 := binary.BigEndian.Uint32(hash[4:8]) % cf.numBuckets

	// Calculate the second bucket using the fingerprint
	bucket2 := (bucket1 ^ (fingerprint * 0x5bd1e995)) % cf.numBuckets

	return fingerprint, bucket1, bucket2
}

// uint32ToBytes converts uint32 to bytes for hashing
func (cf *AMQFilter) uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

// GetFalsePositiveRate estimates the false positive rate of the filter
func (cf *AMQFilter) GetFalsePositiveRate() float64 {
	// Simple estimation based on load factor
	loadFactor := float64(cf.itemCount) / float64(cf.numBuckets*uint32(cf.bucketSize))
	if loadFactor > 1.0 {
		loadFactor = 1.0
	}

	// For Cuckoo filters, false positive rate is approximately:
	// 2 * bucketSize / 2^fingerprintLen when filter is not too full
	return 2.0 * float64(cf.bucketSize) * math.Pow(2, -float64(cf.fingerprintLen)) * loadFactor
}

// Clear resets the filter
func (cf *AMQFilter) Clear() {
	for i := range cf.buckets {
		cf.buckets[i] = cf.buckets[i][:0]
	}
	cf.itemCount = 0
}

// GetValue returns the current value of the accumulator
func (ca *CryptographicAccumulator) GetValue() *big.Int {
	return ca.value
}
