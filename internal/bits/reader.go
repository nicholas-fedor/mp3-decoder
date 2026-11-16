// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package bits

import "encoding/binary"

// Reader reads bits from a byte slice, most significant bit first.
//
// Reads past the end of the bytes give zeros, so a caller that bounds its reads checks Pos.
type Reader struct {
	// data holds the bits.
	data []byte
	// pos is the position of the next bit.
	pos int
}

// Window sizes.
const (
	// MaxRead is the most bits one Peek or Read returns.
	MaxRead = 32
	// windowBytes is the number of bytes each read loads.
	windowBytes = 8
	// windowBits is the number of bits each read loads.
	windowBits = 64
	// byteShift turns a bit position into a byte offset.
	byteShift = 3
	// bitMask keeps the bit offset within a byte.
	bitMask = 7
)

// Peek returns the next bits without moving past them.
//
// Parameters:
//   - n: the number of bits, from 0 to [MaxRead].
//
// Returns:
//   - uint32: the bits, the first in the highest place.
func (r *Reader) Peek(n int) uint32 {
	return uint32(r.window() >> (windowBits - n)) //nolint:gosec // At most 32 bits remain after the shift.
}

// Pos returns the position of the next bit.
//
// Returns:
//   - int: the position in bits from the start of the bytes.
func (r *Reader) Pos() int {
	return r.pos
}

// Read returns the next bits and moves past them.
//
// Parameters:
//   - n: the number of bits, from 0 to [MaxRead].
//
// Returns:
//   - uint32: the bits, the first in the highest place.
func (r *Reader) Read(n int) uint32 {
	value := r.Peek(n)

	r.pos += n

	return value
}

// Reset starts reading data at a bit position.
//
// Parameters:
//   - data: the bytes, which the reader keeps without copying.
//   - pos: the position of the first bit to read.
func (r *Reader) Reset(data []byte, pos int) {
	r.data = data
	r.pos = pos
}

// Skip moves past bits.
//
// Parameters:
//   - n: the number of bits.
func (r *Reader) Skip(n int) {
	r.pos += n
}

// window returns the 64 bits from the next one on, with zeros past the end of the bytes.
//
// Returns:
//   - uint64: the bits, the next one in the highest place.
func (r *Reader) window() uint64 {
	at := r.pos >> byteShift
	if at >= 0 && at+windowBytes <= len(r.data) {
		return binary.BigEndian.Uint64(r.data[at:]) << (r.pos & bitMask)
	}

	var tail [windowBytes]byte

	if at >= 0 && at < len(r.data) {
		copy(tail[:], r.data[at:])
	}

	return binary.BigEndian.Uint64(tail[:]) << (r.pos & bitMask)
}
