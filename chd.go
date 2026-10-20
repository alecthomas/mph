// Package mph is a Go implementation of the compress, hash and displace (CHD)
// minimal perfect hash algorithm.
//
// See http://cmph.sourceforge.net/papers/esa09.pdf for details.
//
// To create and serialize a hash table:
//
//	b := mph.Builder()
//	for k, v := range data {
//		b.Add(k, v)
//	}
//	h, _ := b.Build()
//	w, _ := os.Create("data.idx")
//	b, _ := h.Write(w)
//
// To read from the hash table:
//
//	r, _ := os.Open("data.idx")
//	h, _ := h.Read(r)
//
//	v := h.Get([]byte("some key"))
//	if v == nil {
//	    // Key not found
//	}
//
// MMAP is also indirectly supported, by deserializing from a byte
// slice and slicing the keys and values.
//
// See https://github.com/alecthomas/mph for source.
package mph

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

// CHD hash table lookup.
type CHD struct {
	// Random hash function table.
	r []uint64
	// Array of indices into hash function table r. We assume there aren't
	// more than 2^16 hash functions O_o
	indices []uint16

	// Keys and values each are stored as one large []byte with one of three options of finding offsets into the byte slice:
	// * The V1 on-disk format stores lengths, keys and values all interleaved. Good for locality, but made Mmap() O(n) and used O(n) memory (outside of the mmap).
	// * The V2 on-disk format stores a list of offsets for each key/value. The length is calculated as (offsets[row+1] - offsets[row]). For this to work the number of offsets is one larger than the number of entries.
	// * The V2 on-disk format has a special case for all keys and/or values being exactly the same length. It means we can calculate offsets with multiplication.
	// V1 uses offsets into the full mmap, so keyData and valueData are the same. V2 has separate sections for each.

	keyData            []byte
	valueData          []byte
	keyStartEnds       []dataSlice // v1 only
	valueStartEnds     []dataSlice // v1 only
	keyOffsets         []uint64    // v2 only
	valueOffsets       []uint64    // v2 only
	numEntries         uint64
	allKeysAreLength   uint64 // v2 only
	allValuesAreLength uint64 // v2 only
}

type dataSlice struct {
	start uint64
	end   uint64
}

func hasher(data []byte) uint64 {
	var hash uint64 = 14695981039346656037
	for _, c := range data {
		hash ^= uint64(c)
		hash *= 1099511628211
	}
	return hash
}

// Read a serialized CHD.
func Read(r io.Reader) (*CHD, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Mmap(b)
}

// Mmap creates a new CHD aliasing the CHD structure over an existing byte region (typically mmapped).
func Mmap(b []byte) (*CHD, error) {
	c := &CHD{}

	bi := &sliceReader{b: b}

	// Read vector of hash functions.
	var v2 bool
	rl := bi.ReadInt()
	if rl == 0 {
		v2 = true
		rl = bi.ReadInt()
	}
	c.r = bi.ReadUint64Array(rl)

	// Read hash function indices.
	il := bi.ReadInt()
	c.indices = bi.ReadUint16Array(il)

	c.numEntries = bi.ReadInt()

	if v2 {
		c.allKeysAreLength = bi.ReadInt()
		c.allValuesAreLength = bi.ReadInt()
		if c.allKeysAreLength == 0 {
			c.keyOffsets = bi.ReadUint64Array(c.numEntries + 1)
		}
		if c.allValuesAreLength == 0 {
			c.valueOffsets = bi.ReadUint64Array(c.numEntries + 1)
		}
		if c.allKeysAreLength != 0 {
			c.keyData = bi.Read(c.numEntries * c.allKeysAreLength)
		} else {
			c.keyData = bi.Read(c.keyOffsets[c.numEntries])
		}
		if c.allValuesAreLength != 0 {
			c.valueData = bi.Read(c.numEntries * c.allValuesAreLength)
		} else {
			c.valueData = bi.Read(c.valueOffsets[c.numEntries])
		}
	} else {
		// The v1 format is:
		// file := header record*
		// record := keyLen valueLen [key bytes] [value bytes]
		c.keyData = b
		c.valueData = b
		c.keyStartEnds = make([]dataSlice, c.numEntries)
		c.valueStartEnds = make([]dataSlice, c.numEntries)

		for i := uint64(0); i < c.numEntries; i++ {
			kl := bi.ReadInt()
			vl := bi.ReadInt()
			c.keyStartEnds[i].start = bi.pos
			bi.pos += kl
			c.keyStartEnds[i].end = bi.pos
			c.valueStartEnds[i].start = bi.pos
			bi.pos += vl
			c.valueStartEnds[i].end = bi.pos
		}
	}

	return c, nil
}

func (c *CHD) getKey(row int) []byte {
	if c.allKeysAreLength != 0 {
		return c.keyData[row*int(c.allKeysAreLength) : row*int(c.allKeysAreLength)+int(c.allKeysAreLength)]
	}
	if len(c.keyOffsets) != 0 {
		return c.keyData[c.keyOffsets[row]:c.keyOffsets[row+1]]
	}
	ds := c.keyStartEnds[row]
	return c.keyData[ds.start:ds.end]
}

func (c *CHD) getValue(row int) []byte {
	if c.allValuesAreLength != 0 {
		return c.valueData[row*int(c.allValuesAreLength) : row*int(c.allValuesAreLength)+int(c.allValuesAreLength)]
	}
	if len(c.valueOffsets) != 0 {
		return c.valueData[c.valueOffsets[row]:c.valueOffsets[row+1]]
	}
	ds := c.valueStartEnds[row]
	return c.valueData[ds.start:ds.end]
}

// Get an entry from the hash table.
func (c *CHD) Get(key []byte) []byte {
	r0 := c.r[0]
	h := hasher(key) ^ r0
	i := h % uint64(len(c.indices))
	ri := c.indices[i]
	// This can occur if there were unassigned slots in the hash table.
	if ri >= uint16(len(c.r)) {
		return nil
	}
	r := c.r[ri]
	ti := (h ^ r) % uint64(c.numEntries)
	// fmt.Printf("r[0]=%d, h=%d, i=%d, ri=%d, r=%d, ti=%d\n", c.r[0], h, i, ri, r, ti)
	if bytes.Compare(c.getKey(int(ti)), key) != 0 {
		return nil
	}
	return c.getValue(int(ti))
}

func (c *CHD) Len() int {
	return int(c.numEntries)
}

// Iterate over entries in the hash table.
func (c *CHD) Iterate() *Iterator {
	if c.numEntries == 0 {
		return nil
	}
	return &Iterator{c: c}
}

// Serialize the CHD. The serialized form is conducive to mmapped access. See
// the Mmap function for details.
func (c *CHD) Write(w io.Writer) error {
	write := func(nd ...interface{}) error {
		for _, d := range nd {
			if err := binary.Write(w, binary.LittleEndian, d); err != nil {
				return err
			}
		}
		return nil
	}

	data := []interface{}{
		uint32(len(c.r)), c.r,
		uint32(len(c.indices)), c.indices,
		uint32(c.numEntries),
	}

	if err := write(data...); err != nil {
		return err
	}

	for i := range c.numEntries {
		k := c.getKey(int(i))
		v := c.getValue(int(i))
		if err := write(uint32(len(k)), uint32(len(v))); err != nil {
			return err
		}
		if _, err := w.Write(k); err != nil {
			return err
		}
		if _, err := w.Write(v); err != nil {
			return err
		}
	}
	return nil
}

// Serialize the CHD. The serialized form is conducive to mmapped access. See
// the Mmap function for details.
// WriteV2 emits the V2 on-disk format which can only be parsed by compatible readers.
// It can load in O(1) instead of O(n) and uses O(1) instead of O(n) memory besides the mmap.
// Additionally it optimizes the case where all keys and/or values are exactly the same length.
// If not all keys are the same length, V2 is ~4 bytes per entry larger. Same for values.
func (c *CHD) WriteV2(w io.Writer) error {
	write := func(nd ...interface{}) error {
		for _, d := range nd {
			if err := binary.Write(w, binary.LittleEndian, d); err != nil {
				return err
			}
		}
		return nil
	}

	allKeysAreLength := c.allKeysAreLength
	allValuesAreLength := c.allValuesAreLength

	if allKeysAreLength == 0 && len(c.keyStartEnds) > 0 {
		// Looks like we're writing out a loaded V1 file. Let's see if we can optimize.
		allKeysAreLength = c.keyStartEnds[0].end - c.keyStartEnds[0].start
		for _, ds := range c.keyStartEnds[1:] {
			if allKeysAreLength != ds.end-ds.start {
				allKeysAreLength = 0
				break
			}
		}
	}
	if allValuesAreLength == 0 && len(c.keyStartEnds) > 0 {
		// Looks like we're writing out a loaded V1 file. Let's see if we can optimize.
		allValuesAreLength = c.valueStartEnds[0].end - c.valueStartEnds[0].start
		for _, ds := range c.valueStartEnds[1:] {
			if allValuesAreLength != ds.end-ds.start {
				allValuesAreLength = 0
				break
			}
		}
	}

	data := []interface{}{
		uint32(0), // Indicate V2 format
		uint32(len(c.r)), c.r,
		uint32(len(c.indices)), c.indices,
		uint32(c.numEntries),
		uint32(allKeysAreLength),
		uint32(allValuesAreLength),
	}

	if err := write(data...); err != nil {
		return err
	}

	if allKeysAreLength == 0 {
		if err := writeOffsets(write, c.keyOffsets, c.keyStartEnds); err != nil {
			return err
		}
	}
	if allValuesAreLength == 0 {
		if err := writeOffsets(write, c.valueOffsets, c.valueStartEnds); err != nil {
			return err
		}
	}

	for i := range c.numEntries {
		if _, err := w.Write(c.getKey(int(i))); err != nil {
			return err
		}
	}
	for i := range c.numEntries {
		if _, err := w.Write(c.getValue(int(i))); err != nil {
			return err
		}
	}
	return nil
}

func writeOffsets(write func(nd ...any) error, offsets []uint64, startEnds []dataSlice) error {
	if len(offsets) != 0 {
		for _, n := range offsets {
			if err := write(n); err != nil {
				return err
			}
		}
		return nil
	}
	var offset uint64
	for _, n := range startEnds {
		if err := write(offset); err != nil {
			return err
		}
		offset += n.end - n.start
	}
	return write(offset)
}

type Iterator struct {
	i int
	c *CHD
}

func (c *Iterator) Get() (key []byte, value []byte) {
	return c.c.getKey(c.i), c.c.getValue(c.i)
}

func (c *Iterator) Next() *Iterator {
	c.i++
	if uint64(c.i) >= c.c.numEntries {
		return nil
	}
	return c
}
