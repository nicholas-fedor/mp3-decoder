// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLayer1AudioSizes checks the size of the audio data of each mode, which carries an allocation
// for each subband and channel below the bound and one for both channels from it on, then a
// scalefactor and 12 samples for each allocation that is not 0.
func TestLayer1AudioSizes(t *testing.T) {
	t.Parallel()

	var empty, full Layer1Audio

	for ch := range full.Allocation {
		for sb := range Subbands {
			full.Allocation[ch][sb] = 1
		}
	}

	tests := []struct {
		name  string
		head  Header
		empty int
		full  int
	}{
		// 32 allocations, then 32 scalefactors and 12 times 32 samples of 2 bits.
		{"mono", Header{Version: MPEG1, Layer: Layer1, Mode: Mono}, 128, 128 + 32*6 + 12*32*2},
		// 64 allocations, then 64 scalefactors and 12 times 64 samples.
		{"stereo", Header{Version: MPEG1, Layer: Layer1, Mode: Stereo}, 256, 256 + 64*6 + 12*64*2},
		// 8 allocations below subband 4 and 28 from it, then 64 scalefactors and 12 times 36 samples.
		{"joint stereo from subband 4", Header{Version: MPEG1, Layer: Layer1, Mode: JointStereo}, 144, 144 + 64*6 + 12*36*2},
		// 32 allocations below subband 16 and 16 from it.
		{"joint stereo from subband 16", Header{Version: MPEG1, Layer: Layer1, Mode: JointStereo, Extension: 3}, 192, 192 + 64*6 + 12*48*2},
	}

	for _, tt := range tests {
		assert.Len(t, tt.head.Layer1Audio(&empty), tt.empty/byteBits, tt.name)
		assert.Len(t, tt.head.Layer1Audio(&full), (tt.full+byteBits-1)/byteBits, tt.name)
	}
}

// TestLayer1AudioFields checks where the first fields of a mono frame lie.
func TestLayer1AudioFields(t *testing.T) {
	t.Parallel()

	var fields Layer1Audio

	fields.Allocation[0][0], fields.Allocation[0][1] = 0xe, 0x3
	fields.Scalefactors[0][0], fields.Scalefactors[0][1] = 0x3f, 0x01
	fields.Samples[0][0][0], fields.Samples[0][0][1] = 0x7fff, 0x5

	data := Header{Version: MPEG1, Layer: Layer1, Mode: Mono}.Layer1Audio(&fields)

	// The allocations 1110 0011, then 30 zeros.
	assert.Equal(t, []byte{0xe3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, data[:16])
	// The scalefactors 111111 000001, then the samples 111111111111111 and 0101, then zeros.
	assert.Equal(t, []byte{0xfc, 0x1f, 0xff, 0xea}, data[16:20])
}

// TestRandomLayer1Audio checks that random fields keep to their limits.
func TestRandomLayer1Audio(t *testing.T) {
	t.Parallel()

	fields := RandomLayer1Audio(rand.New(rand.NewPCG(1, 2)), 3)

	for ch := range fields.Allocation {
		for sb, allocation := range fields.Allocation[ch] {
			assert.LessOrEqual(t, allocation, uint32(3))
			assert.LessOrEqual(t, fields.Scalefactors[ch][sb], uint32(62))

			for set := range Layer1Sets {
				assert.Less(t, fields.Samples[set][ch][sb], uint32(1)<<(allocation+1)-1)
			}
		}
	}
}
