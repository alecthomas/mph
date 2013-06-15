// A Go implementation of minimal perfect hashing (MPH).
//
// This package implements the compress, hash and displace (CHD) algorithm
// described here: http://cmph.sourceforge.net/papers/esa09.pdf
//
// See https://github.com/alecthomas/mph for source
package mph

// CHD hash table lookup.
type CHD struct {
	// Random hash function table.
	r []uint64
	// Array of indices into hash function table r
	indices []uint64
	// Final table of values.
	table []*CHDKeyValue
}

func (c *CHD) Get(key []byte) []byte {
	h := CDHHash(key) ^ c.r[0]
	i := h % uint64(len(c.indices))
	j := c.indices[i]
	r := c.r[j]
	k := (h ^ r) % uint64(len(c.table))
	return c.table[k].value
}

func (c *CHD) Iterate() Iterator {
	return &CHDIterator{c: c}
}

type CHDIterator struct {
	i int
	c *CHD
}

func (c *CHDIterator) Get() Entry {
	return c.c.table[c.i]
}

func (c *CHDIterator) Next() Iterator {
	c.i++
	if c.i >= len(c.c.table) {
		return nil
	}
	return c
}
