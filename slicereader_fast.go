//go:build 386 || amd64 || arm
// +build 386 amd64 arm

package mph

import (
	"encoding/binary"

	"github.com/alecthomas/unsafeslice"
)

// Read values and typed vectors from a byte slice without copying where
// possible. This implementation directly references the underlying byte slice
// for array operations, making them essentially zero copy. As the data is
// written in little endian form, this of course means that this will only
// work on little-endian architectures.
type sliceReader struct {
	b   []byte
	pos uint64
}

func (b *sliceReader) Read(size uint64) []byte {
	start := b.pos
	b.pos += size
	return b.b[start:b.pos]
}

func (b *sliceReader) ReadUint64Array(n uint64) []uint64 {
	start := b.pos
	b.pos += n * 8
	return unsafeslice.Uint64SliceFromByteSlice(b.b[start:b.pos])
}

func (b *sliceReader) ReadUint16Array(n uint64) []uint16 {
	start := b.pos
	b.pos += n * 2
	return unsafeslice.Uint16SliceFromByteSlice(b.b[start:b.pos])
}

// Despite returning a uint64, this actually reads a uint32. All table indices
// and lengths are stored as uint32 values.
func (b *sliceReader) ReadInt() uint64 {
	return uint64(binary.LittleEndian.Uint32(b.Read(4)))
}
