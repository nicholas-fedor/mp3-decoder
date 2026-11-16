// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package huffman

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// TestCount1Overrun checks that a quadruple whose bits run past the region's end is left out.
//
// Table B codes 0000 as 1111, and each 1111 here is followed by no signs, so each quadruple takes 4
// bits. With the end at bit 10, the third quadruple ends at bit 12 and is left out.
func TestCount1Overrun(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0xff, 0xff}, 0)

	got := make([]int, 12)
	decoded := Count1(&r, 1, got, 10)

	assert.Equal(t, 8, decoded)
	assert.Equal(t, make([]int, 12), got)
	assert.Equal(t, 12, r.Pos())
}

// TestCount1Full checks that decoding stops when the values are full, before the region's end.
func TestCount1Full(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0xff, 0xff}, 0)

	got := make([]int, 6)
	decoded := Count1(&r, 1, got, 16)

	assert.Equal(t, 4, decoded)
	assert.Equal(t, 4, r.Pos())
}

// TestCount1AtEnd checks that a region already at its end decodes nothing.
func TestCount1AtEnd(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0xff}, 3)

	assert.Zero(t, Count1(&r, 0, make([]int, 4), 3))
	assert.Equal(t, 3, r.Pos())
}

// TestQuadTableB checks table B's rule, by which the code word 0000 is the quadruple 1111.
func TestQuadTableB(t *testing.T) {
	t.Parallel()

	var r bits.Reader

	r.Reset([]byte{0x0f}, 0)
	assert.Equal(t, 0b1111, quad(&r, 1))
	assert.Equal(t, 0b0000, quad(&r, 1))
}
