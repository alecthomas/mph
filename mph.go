// A Go implementation of a minimal perfect hashing algorithm.
//
// This package is the interface. See github.com/alecthomas/mph/chd for an
// implementation of the compress, hash and displace MPH algorithm.
//
// See http://godoc.org/github.com/alecthomas/mph for documentation.
//
package mph

import (
	"io"
)

// A hash table entry.
type Entry interface {
	Key() []byte
	Value() []byte
}

// An iterator over the entries in a hash table.
// for i := m.Iterate(); i != nil; i = i.Next() { v := i.Get(); ... }
type Iterator interface {
	Next() Iterator
	Get() Entry
}

// A hash table reader interface.
type Hash interface {
	Get(key []byte) []byte
	Len() int
	// Iterate over the entries in the hash table.
	Iterate() Iterator
	// Serialize the hash table.
	Write(w io.Writer) error
}
