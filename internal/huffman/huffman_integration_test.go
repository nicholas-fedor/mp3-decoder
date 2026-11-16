// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package huffman_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// pairs lists every pair of values a big_values table codes, each sign of each, and for a table
// with linbits, magnitudes at 15, one above it and at the top of its linbits.
func pairs(table codebook.BigValueTable) []int {
	magnitudes := make([]int, 0, table.Width()+2)
	for magnitude := range table.Width() {
		magnitudes = append(magnitudes, magnitude)
	}

	if table.Linbits() > 0 {
		magnitudes = append(magnitudes, 16, 15+1<<table.Linbits()-1)
	}

	values := make([]int, 0, 8*len(magnitudes)*len(magnitudes))

	for _, x := range magnitudes {
		for _, y := range magnitudes {
			values = append(values, x, y, -x, y, x, -y, -x, -y)
		}
	}

	return values
}

// TestBigValuesRoundTrip encodes every pair of every used big_values table and decodes it back,
// ending on the encoder's last bit.
func TestBigValuesRoundTrip(t *testing.T) {
	t.Parallel()

	for number := range codebook.BigValueTables {
		table, ok := codebook.BigValue(number)
		if !ok {
			continue
		}

		values := pairs(table)

		var w mp3test.BitWriter

		w.BigValues(number, values)

		var r bits.Reader

		r.Reset(w.Bytes(), 0)

		got := make([]int, len(values))

		require.Equal(t, len(values), huffman.BigValues(&r, number, got, w.Len()), "table %d", number)
		require.Equal(t, values, got, "table %d", number)
		assert.Equal(t, w.Len(), r.Pos(), "table %d", number)
	}
}

// TestCount1RoundTrip encodes every quadruple with every sign in both count1 tables and decodes it
// back to the region's end.
func TestCount1RoundTrip(t *testing.T) {
	t.Parallel()

	values := make([]int, 0, 4*codebook.Count1Values*codebook.Count1Values)

	for quad := range codebook.Count1Values {
		for signs := range codebook.Count1Values {
			for i := range 4 {
				value := quad >> (3 - i) & 1
				if signs>>(3-i)&1 == 1 {
					value = -value
				}

				values = append(values, value)
			}
		}
	}

	for table := range codebook.Count1Tables {
		var w mp3test.BitWriter

		w.Count1(table, values)

		var r bits.Reader

		r.Reset(w.Bytes(), 0)

		got := make([]int, len(values))
		decoded := huffman.Count1(&r, table, got, w.Len())

		require.Equal(t, len(values), decoded, "table %d", table)
		require.Equal(t, values, got, "table %d", table)
		assert.Equal(t, w.Len(), r.Pos(), "table %d", table)
	}
}
