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
	// Final table of values.
	mmap   []byte
	keys   []dataSlice
	values []dataSlice
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
	c := &CHD{mmap: b}

	bi := &sliceReader{b: b}

	// Read vector of hash functions.
	rl := bi.ReadInt()
	c.r = bi.ReadUint64Array(rl)

	// Read hash function indices.
	il := bi.ReadInt()
	c.indices = bi.ReadUint16Array(il)

	el := bi.ReadInt()

	c.keys = make([]dataSlice, el)
	c.values = make([]dataSlice, el)

	for i := uint64(0); i < el; i++ {
		kl := bi.ReadInt()
		vl := bi.ReadInt()
		c.keys[i].start = bi.pos
		bi.pos += kl
		c.keys[i].end = bi.pos
		c.values[i].start = bi.pos
		bi.pos += vl
		c.values[i].end = bi.pos
	}

	return c, nil
}

func (c *CHD) slice(s dataSlice) []byte {
	return c.mmap[s.start:s.end]
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
	ti := (h ^ r) % uint64(len(c.keys))
	// fmt.Printf("r[0]=%d, h=%d, i=%d, ri=%d, r=%d, ti=%d\n", c.r[0], h, i, ri, r, ti)
	k := c.keys[ti]
	if bytes.Compare(c.slice(k), key) != 0 {
		return nil
	}
	v := c.values[ti]
	return c.slice(v)
}

func (c *CHD) Len() int {
	return len(c.keys)
}

// Iterate over entries in the hash table.
func (c *CHD) Iterate() *Iterator {
	if len(c.keys) == 0 {
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
		uint32(len(c.keys)),
	}

	if err := write(data...); err != nil {
		return err
	}

	for i := range c.keys {
		k, v := c.keys[i], c.values[i]
		if err := write(uint32(k.end-k.start), uint32(v.end-v.start)); err != nil {
			return err
		}
		if _, err := w.Write(c.slice(k)); err != nil {
			return err
		}
		if _, err := w.Write(c.slice(v)); err != nil {
			return err
		}
	}
	return nil
}

type Iterator struct {
	i int
	c *CHD
}

func (c *Iterator) Get() (key []byte, value []byte) {
	return c.c.slice(c.c.keys[c.i]), c.c.slice(c.c.values[c.i])
}

func (c *Iterator) Next() *Iterator {
	c.i++
	if c.i >= len(c.c.keys) {
		return nil
	}
	return c
}
