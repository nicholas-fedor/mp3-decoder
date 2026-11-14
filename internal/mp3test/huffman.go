// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import "github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"

// escape is the magnitude that linbits extend.
const escape = 15

// BigValues writes pairs of spectral values in a big_values table's code.
//
// Each pair is the code word of its magnitudes, then the linbits and the sign of x, then those of
// y. In a table with linbits, a magnitude of 15 or more is coded as 15 with the rest in its
// linbits. An unused table writes nothing.
//
// Parameters:
//   - table: the table number, from 0 to 31.
//   - values: the values, two for each pair, within the table's range.
func (w *BitWriter) BigValues(table int, values []int) {
	codes, ok := codebook.BigValue(table)
	if !ok {
		return
	}

	for at := 0; at+1 < len(values); at += 2 {
		x, y := abs(values[at]), abs(values[at+1])
		code := codes.Code(min(x, escape), min(y, escape))

		w.Write(code.Bits, code.Len)
		w.value(values[at], codes.Linbits())
		w.value(values[at+1], codes.Linbits())
	}
}

// Count1 writes quadruples of spectral values in a count1 table's code.
//
// Each quadruple is the code word of its magnitudes, then the sign of each nonzero value, v first.
//
// Parameters:
//   - table: the table number, 0 for table A or 1 for table B.
//   - values: the values, four for each quadruple, each from -1 to 1.
func (w *BitWriter) Count1(table int, values []int) {
	codes, ok := codebook.Count1(table)
	if !ok {
		return
	}

	for at := 0; at+3 < len(values); at += 4 {
		quad := 0
		for _, value := range values[at : at+4] {
			quad = quad<<1 | abs(value)
		}

		code := codes.Code(quad)
		w.Write(code.Bits, code.Len)

		for _, value := range values[at : at+4] {
			w.value(value, 0)
		}
	}
}

// value writes a value's linbits and sign after its code word.
//
// Parameters:
//   - value: the value.
//   - linbits: the number of linbits the table has.
func (w *BitWriter) value(value, linbits int) {
	magnitude := abs(value)

	if linbits > 0 && magnitude >= escape {
		w.Write(uint32(magnitude-escape), linbits) //nolint:gosec // Within the linbits.
	}

	if value < 0 {
		w.Write(1, 1)
	} else if value > 0 {
		w.Write(0, 1)
	}
}

// abs returns a value's magnitude.
//
// Parameters:
//   - value: the value.
//
// Returns:
//   - int: the magnitude.
func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
