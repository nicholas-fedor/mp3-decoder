// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package huffman

import "github.com/nicholas-fedor/mp3-decoder/internal/bits"

// escape is the magnitude that linbits extend.
const escape = 15

// BigValues decodes the pairs of spectral values of one big_values region.
//
// Each pair is a code word of the region's table, then the linbits and the sign of x, then those
// of y. The unused tables 4 and 14 decode as table 0, whose pairs are zeros coded in no bits.
// Decoding stops when dst is full or at the end of the granule's Huffman data, and a pair that
// starts before the end is decoded whole. Pairs coded in no bits never reach the end, so table 0
// fills dst.
//
// Parameters:
//   - stream: the bitstream, at the region's first code word.
//   - table: the table number that table_select gives, from 0 to 31.
//   - dst: where the values go, two for each pair.
//   - end: the position of the bit after the granule's Huffman data.
//
// Returns:
//   - int: the number of values decoded, a multiple of 2.
func BigValues(stream *bits.Reader, table int, dst []int, end int) int {
	ref := bigValueLookups[table]

	if ref.root == 0 {
		decoded := len(dst) &^ 1
		clear(dst[:decoded])

		return decoded
	}

	decoded := 0

	for ; decoded+1 < len(dst) && stream.Pos() < end; decoded += 2 {
		x, y := readCode(&lookupEntries, ref, stream)

		dst[decoded] = signed(stream, extend(stream, x, ref.linbits))
		dst[decoded+1] = signed(stream, extend(stream, y, ref.linbits))
	}

	return decoded
}

// extend adds a magnitude's linbits.
//
// Parameters:
//   - stream: the bitstream, at the linbits.
//   - magnitude: the magnitude the code word gives.
//   - linbits: the number of linbits the table has.
//
// Returns:
//   - int: the magnitude, extended when it is 15 and the table has linbits.
func extend(stream *bits.Reader, magnitude int, linbits uint8) int {
	if magnitude == escape && linbits > 0 {
		return magnitude + int(stream.Read(int(linbits)))
	}

	return magnitude
}

// signed applies a magnitude's sign bit.
//
// Parameters:
//   - stream: the bitstream, at the sign bit, which a magnitude of 0 does not have.
//   - magnitude: the magnitude.
//
// Returns:
//   - int: the value, negative when the sign bit is 1.
func signed(stream *bits.Reader, magnitude int) int {
	if magnitude != 0 && stream.Read(1) == 1 {
		return -magnitude
	}

	return magnitude
}
