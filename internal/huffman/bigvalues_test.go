// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// TestBigValuesNoBits checks that table 0 and the unused tables 4 and 14 decode zeros and read no
// bits, even with the reader at the end of the Huffman data.
func TestBigValuesNoBits(t *testing.T) {
	t.Parallel()

	for _, table := range []int{0, 4, 14} {
		var r bits.Reader

		r.Reset([]byte{0xff, 0xff}, 0)

		got := []int{9, 9, 9, 9}

		assert.Equal(t, 4, BigValues(&r, table, got, 0), "table %d", table)
		assert.Equal(t, []int{0, 0, 0, 0}, got, "table %d", table)
		assert.Zero(t, r.Pos(), "table %d", table)
	}
}

// TestBigValuesEnd checks that decoding stops at the end of the Huffman data, and that a pair that
// starts before the end is decoded whole.
//
// Table 1 codes the pair (0, 0) as 1 and (1, 0) as 01. The bits 1 1 01 0 hold two pairs of zeros,
// then (1, 0) from bit 2 with its sign at bit 4. With the end at bit 3, that pair is the last.
func TestBigValuesEnd(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0b1101_0000}, 0)

	got := []int{9, 9, 9, 9, 9, 9, 9, 9}

	assert.Equal(t, 6, BigValues(&r, 1, got, 3))
	assert.Equal(t, []int{0, 0, 0, 0, 1, 0, 9, 9}, got)
	assert.Equal(t, 5, r.Pos())
}

// TestExtend covers a magnitude of 15 in tables with and without linbits, and a smaller one.
func TestExtend(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0b1011_0000}, 0)
	assert.Equal(t, 15+0b1011, extend(&r, 15, 4))
	assert.Equal(t, 4, r.Pos())

	r.Reset([]byte{0xff}, 0)
	assert.Equal(t, 15, extend(&r, 15, 0))
	assert.Equal(t, 14, extend(&r, 14, 4))
	assert.Zero(t, r.Pos())
}

// TestSigned covers both sign bits and a zero, which has none.
func TestSigned(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0b10_000000}, 0)
	assert.Equal(t, -3, signed(&r, 3))
	assert.Equal(t, 3, signed(&r, 3))
	assert.Zero(t, signed(&r, 0))
	assert.Equal(t, 2, r.Pos())
}
