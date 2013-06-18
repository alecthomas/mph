package mph

import (
	"github.com/riobard/go-mmap"
	"os"
	"testing"
)

func BenchmarkWikipediaMmappedIndex(b *testing.B) {
	f, _ := os.Open("wikipedia.chd")
	defer f.Close()
	m, err := mmap.Map(f, 0, 15778920, mmap.PROT_READ, 0)
	if err != nil {
		panic(err)
	}
	h, _ := Mmap(m)
	keys := [][]byte{}
	for i := h.Iterate(); i != nil; i = i.Next() {
		kv := i.Get()
		keys = append(keys, kv.Key())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Get(keys[i%len(keys)])
	}
}
