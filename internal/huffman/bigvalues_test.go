// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// TestBigValuesNoBits checks that table 0 and the unused tables 4 and 14 decode zeros and read no
// bits.
func TestBigValuesNoBits(t *testing.T) {
	t.Parallel()

	for _, table := range []int{0, 4, 14} {
		var r bits.Reader

		r.Reset([]byte{0xff, 0xff}, 0)

		got := []int{9, 9, 9, 9}
		BigValues(&r, table, got)

		assert.Equal(t, []int{0, 0, 0, 0}, got, "table %d", table)
		assert.Zero(t, r.Pos(), "table %d", table)
	}
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
