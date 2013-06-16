package mph

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
	// Iterate over the entries in the hash table.
	Iterate() Iterator
	// Serialize the hash table.
	Marshal() ([]byte, error)
}
