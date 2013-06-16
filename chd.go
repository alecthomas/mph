// A Go implementation of minimal perfect hashing (MPH).
//
// This package implements the compress, hash and displace (CHD) algorithm
// described here: http://cmph.sourceforge.net/papers/esa09.pdf
//
// See https://github.com/alecthomas/mph for source
package mph

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
)

// CHD hash table lookup.
type CHD struct {
	// Random hash function table.
	r []uint64
	// Array of indices into hash function table r
	indices []uint64
	// Final table of values.
	table []*CHDKeyValue
}

// Read a protobuf serialized CHD.
func UnmarshalCHD(b []byte) (*CHD, error) {
	pb := &CHDProto{}
	if err := proto.Unmarshal(b, pb); err != nil {
		return nil, err
	}
	c := &CHD{
		r:       pb.GetR(),
		indices: pb.GetIndicies(),
	}
	for _, kv := range pb.GetTable() {
		c.table = append(c.table, &CHDKeyValue{key: kv.GetKey(), value: kv.GetValue()})
	}
	return c, nil
}

func (c *CHD) Get(key []byte) []byte {
	h := CDHHash(key) ^ c.r[0]
	i := h % uint64(len(c.indices))
	if i >= uint64(len(c.indices)) {
		return nil
	}
	ri := c.indices[i]
	if ri >= uint64(len(c.r)) {
		return nil
	}
	r := c.r[ri]
	ti := (h ^ r) % uint64(len(c.table))
	if ti >= uint64(len(c.table)) {
		return nil
	}
	e := c.table[ti]
	if bytes.Compare(e.key, key) != 0 {
		return nil
	}
	return e.value
}

func (c *CHD) Iterate() Iterator {
	if len(c.table) == 0 {
		return nil
	}
	return &CHDIterator{c: c}
}

// Serialize the CHD as a protobuf. See chd.proto for details.
func (c *CHD) Serialize() ([]byte, error) {
	table := []*CHDProto_KeyValue{}
	for _, kv := range c.table {
		table = append(table, &CHDProto_KeyValue{Key: kv.key, Value: kv.value})
	}
	pb := &CHDProto{
		R:        c.r,
		Indicies: c.indices,
		Table:    table,
	}
	return proto.Marshal(pb)
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
