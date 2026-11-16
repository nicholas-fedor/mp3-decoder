// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// granuleBits returns the width of one granule's side information for one channel.
//
// Either branch of window switching takes 22 bits.
func granuleBits(lay layout) int {
	bits := part23Bits + bigValuesBits + globalGainBits + lay.compress + 1 + 22 + 1 + 1
	if lay.preflag {
		bits++
	}

	return bits
}

// TestLayoutSizes checks that each layout's widths add up to the published side information sizes:
// 17 and 32 bytes for MPEG-1 mono and stereo, and 9 and 17 for MPEG-2 and MPEG-2.5.
func TestLayoutSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lay      layout
		channels int
		bytes    int
	}{
		{"MPEG-1 mono", mpeg1Layout, 1, 17},
		{"MPEG-1 stereo", mpeg1Layout, 2, 32},
		{"MPEG-2 mono", lsfLayout, 1, 9},
		{"MPEG-2 stereo", lsfLayout, 2, 17},
	}

	for _, tt := range tests {
		private := tt.lay.privateStereo
		if tt.channels == 1 {
			private = tt.lay.privateMono
		}

		bits := tt.lay.begin + private + tt.lay.granules*tt.channels*granuleBits(tt.lay)
		if tt.lay.scfsi {
			bits += 4 * tt.channels
		}

		assert.Equal(t, tt.bytes*8, bits, tt.name)
	}

	assert.Equal(t, 22, blockTypeBits+1+switchedRegions*tableSelectBits+3*subblockGainBits)
	assert.Equal(t, 22, 3*tableSelectBits+region0Bits+region1Bits)
}
