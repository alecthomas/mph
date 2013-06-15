package mph

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"testing"
)

func TestCDHBuilder(t *testing.T) {
	d := map[string]string{
		"one":   "1",
		"two":   "2",
		"three": "3",
		"four":  "4",
		"five":  "5",
		"six":   "6",
		"seven": "7",
	}
	b := NewCHDBuilder()
	for k, v := range d {
		b.Add([]byte(k), []byte(v))
	}
	c, err := b.Build()
	assert.NoError(t, err)
	assert.Equal(t, 7, len(c.table))
	for k, v := range d {
		assert.Equal(t, []byte(v), c.Get([]byte(k)))
	}
}

func BenchmarkBuiltinMap(b *testing.B) {
	keys := []string{}
	d := map[string]string{}
	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("%d", i)
		d[k] = k
		keys = append(keys, k)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			_ = d[k]
		}
	}
}

func BenchmarkCDH(b *testing.B) {
	keys := [][]byte{}
	mph := NewCHDBuilder()
	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("%d", i)
		keys = append(keys, []byte(k))
		mph.Add([]byte(k), []byte(k))
	}
	h, _ := mph.Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			h.Get(k)
		}
	}
}
