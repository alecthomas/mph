//go:build !386 && !amd64 && !arm
// +build !386,!amd64,!arm

package mph

import (
	"encoding/binary"
)

// Read values and typed vectors from a byte slice without copying where possible.
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
	buf := b.Read(n * 8)
	out := make([]uint64, n)
	for i := 0; i < len(buf); i += 8 {
		out[i>>3] = binary.LittleEndian.Uint64(buf[i : i+8])
	}
	return out
}

func (b *sliceReader) ReadUint16Array(n uint64) []uint16 {
	buf := b.Read(n * 2)
	out := make([]uint16, n)
	for i := 0; i < len(buf); i += 2 {
		out[i>>1] = binary.LittleEndian.Uint16(buf[i : i+2])
	}
	return out
}

func (b *sliceReader) ReadInt() uint64 {
	return uint64(binary.LittleEndian.Uint32(b.Read(4)))
}
