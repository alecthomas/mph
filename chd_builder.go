package mph

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type chdHasher struct {
	r       []uint64
	size    uint64
	buckets uint64
	rand    *rand.Rand
}

type bucket struct {
	index  uint64
	keys   [][]byte
	values [][]byte
}

func (b *bucket) String() string {
	a := "bucket{"
	for _, k := range b.keys {
		a += string(k) + ", "
	}
	return a + "}"
}

// Intermediate data structure storing buckets + outer hash index.
type bucketVector []bucket

func (b bucketVector) Len() int           { return len(b) }
func (b bucketVector) Less(i, j int) bool { return len(b[i].keys) > len(b[j].keys) }
func (b bucketVector) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// Build a new CDH MPH.
type CHDBuilder struct {
	keys   [][]byte
	values [][]byte
}

// Create a new CHD hash table builder.
func Builder() *CHDBuilder {
	return &CHDBuilder{}
}

// Add a key and value to the hash table.
func (b *CHDBuilder) Add(key []byte, value []byte) {
	b.keys = append(b.keys, key)
	b.values = append(b.values, value)
}

// Try to find a hash function that does not cause collisions with table, when
// applied to the keys in the bucket.
func tryHash(hasher *chdHasher, seen map[uint64]bool, keys [][]byte, values [][]byte, indices []uint16, bucket *bucket, ri uint16, r uint64) bool {
	// Track duplicates within this bucket.
	duplicate := make(map[uint64]bool)
	// Make hashes for each entry in the bucket.
	hashes := make([]uint64, len(bucket.keys))
	for i, k := range bucket.keys {
		h := hasher.Table(r, k)
		hashes[i] = h
		if seen[h] {
			return false
		}
		if duplicate[h] {
			return false
		}
		duplicate[h] = true
	}

	// Update seen hashes
	for _, h := range hashes {
		seen[h] = true
	}

	// Add the hash index.
	indices[bucket.index] = ri

	// Update the the hash table.
	for i, h := range hashes {
		keys[h] = bucket.keys[i]
		values[h] = bucket.values[i]
	}
	return true
}

func (b *CHDBuilder) Build() (*CHD, error) {
	n := uint64(len(b.keys))
	m := n / 2
	if m == 0 {
		m = 1
	}

	keys := make([][]byte, n)
	values := make([][]byte, n)
	hasher := newCHDHasher(n, m)
	buckets := make(bucketVector, m)
	indices := make([]uint16, m)
	// An extra check to make sure we don't use an invalid index
	for i := range indices {
		indices[i] = ^uint16(0)
	}
	// Have we seen a hash before?
	seen := make(map[uint64]bool)
	// Used to ensure there are no duplicate keys.
	duplicates := make(map[string]bool)

	for i := range b.keys {
		key := b.keys[i]
		value := b.values[i]
		k := string(key)
		if duplicates[k] {
			return nil, errors.New("duplicate key " + k)
		}
		duplicates[k] = true
		oh := hasher.HashIndexFromKey(key)

		buckets[oh].index = oh
		buckets[oh].keys = append(buckets[oh].keys, key)
		buckets[oh].values = append(buckets[oh].values, value)
	}

	// Order buckets by size (retaining the hash index)
	collisions := 0
	sort.Sort(buckets)
nextBucket:
	for i, bucket := range buckets {
		if len(bucket.keys) == 0 {
			continue
		}

		// Check existing hash functions.
		for ri, r := range hasher.r {
			if tryHash(hasher, seen, keys, values, indices, &bucket, uint16(ri), r) {
				continue nextBucket
			}
		}

		// Keep trying new functions until we get one that does not collide.
		// The number of retries here is very high to allow a very high
		// probability of not getting collisions.
		for i := 0; i < 10000000; i++ {
			if i > collisions {
				collisions = i
			}
			ri, r := hasher.Generate()
			if tryHash(hasher, seen, keys, values, indices, &bucket, ri, r) {
				hasher.Add(r)
				continue nextBucket
			}
		}

		// Failed to find a hash function with no collisions.
		return nil, fmt.Errorf(
			"failed to find a collision-free hash function after ~10000000 attempts, for bucket %d/%d with %d entries: %s",
			i, len(buckets), len(bucket.keys), &bucket)
	}

	// println("max bucket collisions:", collisions)
	// println("keys:", len(table))
	// println("hash functions:", len(hasher.r))

	keylist := make([]dataSlice, len(b.keys))
	valuelist := make([]dataSlice, len(b.values))
	var buf bytes.Buffer
	for i, k := range keys {
		keylist[i].start = uint64(buf.Len())
		buf.Write(k)
		keylist[i].end = uint64(buf.Len())
		valuelist[i].start = uint64(buf.Len())
		buf.Write(values[i])
		valuelist[i].end = uint64(buf.Len())
	}

	return &CHD{
		r:       hasher.r,
		indices: indices,
		mmap:    buf.Bytes(),
		keys:    keylist,
		values:  valuelist,
	}, nil
}

func newCHDHasher(size uint64, buckets uint64) *chdHasher {
	rs := rand.NewSource(time.Now().UnixNano())
	c := &chdHasher{size: size, buckets: buckets, rand: rand.New(rs)}
	c.Add(c.random())
	return c
}

func (c *chdHasher) random() uint64 {
	return (uint64(c.rand.Uint32()) << 32) | uint64(c.rand.Uint32())
}

// Hash index from key.
func (h *chdHasher) HashIndexFromKey(b []byte) uint64 {
	return (hasher(b) ^ h.r[0]) % h.buckets
}

// Table hash from random value and key. Generate() returns these random values.
func (h *chdHasher) Table(r uint64, b []byte) uint64 {
	return (hasher(b) ^ h.r[0] ^ r) % h.size
}

func (c *chdHasher) Generate() (uint16, uint64) {
	return c.Len(), c.random()
}

// Add a random value generated by Generate().
func (c *chdHasher) Add(r uint64) {
	c.r = append(c.r, r)
}

func (c *chdHasher) Len() uint16 {
	return uint16(len(c.r))
}

func (h *chdHasher) String() string {
	return fmt.Sprintf("chdHasher{size: %d, buckets: %d, r: %v}", h.size, h.buckets, h.r)
}
