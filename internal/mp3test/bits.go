// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

// BitWriter writes bits, most significant bit first.
type BitWriter struct {
	// data holds the bytes written so far.
	data []byte
	// bits is the number of bits written.
	bits int
}

// byteBits is the number of bits in a byte.
const byteBits = 8

// Bytes returns the bits written, padded with zeros to a whole byte.
//
// Returns:
//   - []byte: the bytes, which the writer keeps writing into.
func (w *BitWriter) Bytes() []byte {
	return w.data
}

// Len returns the number of bits written.
//
// Returns:
//   - int: the number of bits.
func (w *BitWriter) Len() int {
	return w.bits
}

// Write appends the low n bits of value, the highest first.
//
// Parameters:
//   - value: the bits.
//   - n: the number of bits, from 0 to 32.
func (w *BitWriter) Write(value uint32, n int) {
	for shift := n - 1; shift >= 0; shift-- {
		if w.bits%byteBits == 0 {
			w.data = append(w.data, 0)
		}

		if value>>shift&1 == 1 {
			w.data[len(w.data)-1] |= 1 << (byteBits - 1 - w.bits%byteBits)
		}

		w.bits++
	}
}
