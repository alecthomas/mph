package mph

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"testing"
)

var (
	sampleData = map[string]string{
		"one":   "1",
		"two":   "2",
		"three": "3",
		"four":  "4",
		"five":  "5",
		"six":   "6",
		"seven": "7",
	}
)

func TestCDHBuilder(t *testing.T) {
	b := NewCHDBuilder()
	for k, v := range sampleData {
		b.Add([]byte(k), []byte(v))
	}
	c, err := b.Build()
	assert.NoError(t, err)
	assert.Equal(t, 7, len(c.table))
	for k, v := range sampleData {
		assert.Equal(t, []byte(v), c.Get([]byte(k)))
	}
	assert.Nil(t, c.Get([]byte("monkey")))
}

func TestCDHSerialization(t *testing.T) {
	cb := NewCHDBuilder()
	for k, v := range sampleData {
		cb.Add([]byte(k), []byte(v))
	}
	m, err := cb.Build()
	assert.NoError(t, err)
	b, err := m.Serialize()
	assert.NoError(t, err)
	n, err := UnmarshalCHD(b)
	assert.Equal(t, n.r, m.r)
	assert.Equal(t, n.indices, m.indices)
	assert.Equal(t, n.table, m.table)
	for k, v := range sampleData {
		assert.Equal(t, []byte(v), n.Get([]byte(k)))
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
