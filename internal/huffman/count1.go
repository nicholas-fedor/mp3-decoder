// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import "github.com/nicholas-fedor/mp3-decoder/internal/bits"

// count1 tables.
const (
	// count1TableA is the number count1table_select gives table A.
	count1TableA = 0
	// quadruple is the number of values one count1 code word codes.
	quadruple = 4
	// count1BBits is the length of every code word of count1 table B, which is its value's bits
	// inverted.
	count1BBits = 4
	// count1BMask keeps a count1 table B value.
	count1BMask = 1<<count1BBits - 1
)

// Count1 decodes the quadruples of spectral values of one count1 region.
//
// Each quadruple is a code word of the region's table, then the sign of each nonzero value, v
// first. Decoding stops at the region's end or when dst is full. A quadruple whose bits run past
// the end is left out, and the reader stays past it.
//
// Parameters:
//   - stream: the bitstream, at the region's first code word.
//   - table: the table number that count1table_select gives, 0 for table A or 1 for table B.
//   - dst: where the values go, four for each quadruple.
//   - end: the position of the bit after the region.
//
// Returns:
//   - int: the number of values decoded, a multiple of 4.
func Count1(stream *bits.Reader, table int, dst []int, end int) int {
	decoded := 0

	for decoded+quadruple <= len(dst) && stream.Pos() < end {
		values := quad(stream, table)

		for i := range quadruple {
			dst[decoded+i] = signed(stream, values>>(quadruple-1-i)&1)
		}

		if stream.Pos() > end {
			clear(dst[decoded : decoded+quadruple])

			break
		}

		decoded += quadruple
	}

	return decoded
}

// quad reads one count1 code word.
//
// Parameters:
//   - stream: the bitstream, at the code word.
//   - table: the table number, 0 for table A or 1 for table B.
//
// Returns:
//   - int: the quadruple, v<<3 | w<<2 | x<<1 | y.
func quad(stream *bits.Reader, table int) int {
	if table == count1TableA {
		_, values := readCode(&lookupEntries, count1ALookup, stream)

		return values
	}

	return int(^stream.Read(count1BBits) & count1BMask)
}
