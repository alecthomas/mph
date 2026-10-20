package mph

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

var (
	words [][]byte
)

func init() {
	f, err := os.Open("testdata/words")
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		words = append(words, line)
	}
}

func cmpKeys(t *testing.T, cl, cr *CHD) {
	t.Helper()
	assert.Equal(t, cl.numEntries, cr.numEntries)
	for i := 0; int(cl.numEntries) > i; i++ {
		assert.Equal(t, cl.getKey(i), cr.getKey(i))
	}
}

func cmpValues(t *testing.T, cl, cr *CHD) {
	t.Helper()
	assert.Equal(t, cl.numEntries, cr.numEntries)
	for i := 0; int(cl.numEntries) > i; i++ {
		assert.Equal(t, cl.getValue(i), cr.getValue(i))
	}
}

func TestCHDBuilder(t *testing.T) {
	b := Builder()
	for k, v := range sampleData {
		b.Add([]byte(k), []byte(v))
	}
	c, err := b.Build()
	assert.NoError(t, err)
	assert.Equal(t, 7, int(c.numEntries))
	for k, v := range sampleData {
		assert.Equal(t, []byte(v), c.Get([]byte(k)))
	}
	assert.Nil(t, c.Get([]byte("monkey")))
}

func TestCHDSerialization(t *testing.T) {
	write := func(c *CHD, w io.Writer) error {
		return c.Write(w)
	}
	f := func(t *testing.T) {
		cb := Builder()
		for _, v := range words {
			cb.Add([]byte(v), []byte(v))
		}
		m, err := cb.Build()
		assert.NoError(t, err)
		w := &bytes.Buffer{}
		err = write(m, w)
		assert.NoError(t, err)

		n, err := Mmap(w.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, n.r, m.r)
		assert.Equal(t, n.indices, m.indices)
		cmpKeys(t, n, m)
		cmpValues(t, n, m)
		for _, v := range words {
			assert.Equal(t, []byte(v), n.Get([]byte(v)))
		}
	}
	t.Run("Write", f)
	write = func(c *CHD, w io.Writer) error {
		return c.WriteV2(w)
	}
	t.Run("WriteV2", f)
}

func TestCHDSerialization_empty(t *testing.T) {
	write := func(c *CHD, w io.Writer) error {
		return c.Write(w)
	}
	f := func(t *testing.T) {
		cb := Builder()
		m, err := cb.Build()
		assert.NoError(t, err)
		w := &bytes.Buffer{}
		err = write(m, w)
		assert.NoError(t, err)

		n, err := Mmap(w.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, n.r, m.r)
		assert.Equal(t, n.indices, m.indices)
		cmpKeys(t, n, m)
		cmpValues(t, n, m)
	}
	t.Run("Write", f)
	write = func(c *CHD, w io.Writer) error {
		return c.WriteV2(w)
	}
	t.Run("WriteV2", f)
}

func TestCHDSerialization_one(t *testing.T) {
	write := func(c *CHD, w io.Writer) error {
		return c.Write(w)
	}
	f := func(t *testing.T) {
		cb := Builder()
		cb.Add([]byte("k"), []byte("v"))
		m, err := cb.Build()
		assert.NoError(t, err)
		w := &bytes.Buffer{}
		err = write(m, w)
		assert.NoError(t, err)

		n, err := Mmap(w.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, n.r, m.r)
		assert.Equal(t, n.indices, m.indices)
		cmpKeys(t, n, m)
		cmpValues(t, n, m)
	}
	t.Run("Write", f)
	write = func(c *CHD, w io.Writer) error {
		return c.WriteV2(w)
	}
	t.Run("WriteV2", f)
}

func BenchmarkBuiltinMap(b *testing.B) {
	keys := []string{}
	d := map[string]string{}
	for _, bk := range words {
		k := string(bk)
		d[k] = k
		keys = append(keys, k)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d[keys[i%len(keys)]]
	}
}

func BenchmarkCHD(b *testing.B) {
	keys := [][]byte{}
	mph := Builder()
	for _, k := range words {
		keys = append(keys, k)
		mph.Add(k, k)
	}
	h, _ := mph.Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Get(keys[i%len(keys)])
	}
}
