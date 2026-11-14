// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"
)

// Bit costs of the longest pair and quadruple.
const (
	// maxPairBits is the most bits one pair takes.
	//
	// That is the longest code word, then 13 linbits and a sign for each value.
	maxPairBits = maxLen + 2*(13+1)
	// maxQuadBits is the most bits one quadruple takes.
	//
	// That is the longest code word of table A, then four signs.
	maxQuadBits = 6 + quadruple
)

// FuzzBigValues decodes pairs from any bits in any table and checks that each value lies within
// the table's range and that the reader moves no further than the pairs can take.
//
// The seed corpus holds no bits, ones in table 1 and random-looking bits in tables 13 and 24.
func FuzzBigValues(f *testing.F) {
	f.Add([]byte{}, uint8(0), uint8(4))
	f.Add([]byte{0xff, 0xff}, uint8(1), uint8(8))
	f.Add([]byte{0x93, 0x5a, 0x17, 0xc4, 0x00, 0x81}, uint8(13), uint8(10))
	f.Add([]byte{0x00, 0x00, 0x00, 0x7f, 0xff, 0x12}, uint8(24), uint8(10))

	f.Fuzz(func(t *testing.T, data []byte, number, pairs uint8) {
		table := int(number) % codebook.BigValueTables
		limit := 0

		if codes, ok := codebook.BigValue(table); ok {
			limit = codes.Width() - 1
			if codes.Linbits() > 0 {
				limit = escape + 1<<codes.Linbits() - 1
			}
		}

		var r bits.Reader

		r.Reset(data, 0)

		values := make([]int, int(pairs)%64*2)
		BigValues(&r, table, values)

		for _, value := range values {
			require.LessOrEqual(t, max(value, -value), limit, "table %d", table)
		}

		assert.LessOrEqual(t, r.Pos(), len(values)/2*maxPairBits)
	})
}

// FuzzCount1 decodes quadruples from any bits to any end and checks that each value is -1, 0 or 1,
// that whole quadruples come back, and that the reader stops within one quadruple past the end.
//
// The seed corpus holds no bits, ones in table B and random-looking bits in table A.
func FuzzCount1(f *testing.F) {
	f.Add([]byte{}, uint8(0), uint16(0))
	f.Add([]byte{0xff, 0xff}, uint8(1), uint16(10))
	f.Add([]byte{0x93, 0x5a, 0x17, 0xc4}, uint8(0), uint16(29))

	f.Fuzz(func(t *testing.T, data []byte, number uint8, end uint16) {
		var r bits.Reader

		r.Reset(data, 0)

		values := make([]int, 64)
		decoded := Count1(&r, int(number)%codebook.Count1Tables, values, int(end))

		require.Zero(t, decoded%quadruple)

		for _, value := range values {
			require.Contains(t, []int{-1, 0, 1}, value)
		}

		assert.Less(t, r.Pos(), max(int(end), 0)+maxQuadBits)
	})
}
