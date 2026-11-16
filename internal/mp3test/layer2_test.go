// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLayer2AudioGrouped checks a mono frame of one subband whose class of 3 steps writes each
// granule's three codes as one word of 5 bits, after the allocation, the selection and the one
// scalefactor the selection writes.
func TestLayer2AudioGrouped(t *testing.T) {
	t.Parallel()

	fields := Layer2Audio{Table: []Layer2Subband{{Steps: []int{3, 7, 15}, Width: 4}}}
	fields.Allocation[0][0], fields.Selections[0][0] = 1, 2
	fields.Scalefactors[0][0] = [Layer2Parts]uint32{5, 5, 5}

	for set := 0; set < Layer2Sets; set += 3 {
		fields.Samples[set][0][0], fields.Samples[set+1][0][0] = 1, 2
	}

	data := Header{Version: MPEG1, Layer: Layer2, Mode: Mono}.Layer2Audio(&fields)

	// 0001, 10 and 000101, then the word 1 + 3*2 = 7 as 00111 twelve times.
	assert.Equal(t, []byte{0x18, 0x53, 0x9c, 0xe7, 0x39, 0xce, 0x73, 0x9c, 0xe7}, data)
}

// TestLayer2AudioStereo checks a stereo frame of two subbands of 2-bit allocations, in which each
// channel codes one subband. The first codes 7 steps in 3 bits with the selection that writes the
// first two scalefactors, and the second codes 3 steps in a word with the one that writes the
// first and the third.
func TestLayer2AudioStereo(t *testing.T) {
	t.Parallel()

	fields := Layer2Audio{Table: []Layer2Subband{{Steps: []int{3, 7, 15}, Width: 2}, {Steps: []int{3, 5, 9}, Width: 2}}}
	fields.Allocation[0][0], fields.Allocation[1][1] = 2, 1
	fields.Selections[0][0], fields.Selections[1][1] = 3, 1
	fields.Scalefactors[0][0] = [Layer2Parts]uint32{1, 2, 2}
	fields.Scalefactors[1][1] = [Layer2Parts]uint32{3, 3, 4}

	for set := 0; set < Layer2Sets; set += 3 {
		fields.Samples[set][0][0], fields.Samples[set+1][0][0], fields.Samples[set+2][0][0] = 6, 5, 4
		fields.Samples[set][1][1], fields.Samples[set+1][1][1] = 2, 1
	}

	data := Header{Version: MPEG1, Layer: Layer2, Mode: JointStereo, Extension: 0}.Layer2Audio(&fields)

	// 10 00 00 01, then 11 01, then the scalefactors 1, 2, 3 and 4, then the codes 110 101 100 and
	// the word 2 + 3*1 = 5 as 00101 for each granule, 204 bits in all.
	assert.Len(t, data, 26)
	assert.Equal(t, []byte{0x81, 0xd0, 0x42, 0x0c}, data[:4])
}

// TestLayer2AudioShared checks that from the bound on a stereo frame writes one allocation and one
// set of codes for both channels, and a selection and scalefactors for each.
func TestLayer2AudioShared(t *testing.T) {
	t.Parallel()

	table := make([]Layer2Subband, 8)
	for sb := range table {
		table[sb] = Layer2Subband{Steps: []int{3, 5, 7}, Width: 2}
	}

	fields := Layer2Audio{Table: table}
	fields.Allocation[0][6] = 3

	shared := Header{Version: MPEG1, Layer: Layer2, Mode: JointStereo}.Layer2Audio(&fields)
	apart := Header{Version: MPEG1, Layer: Layer2, Mode: Stereo}.Layer2Audio(&fields)

	// With the bound at 4, the frame writes 24 bits of allocations. Both channels take subband 6,
	// with two selections and six scalefactors in 40 bits, then 12 granules of three 3-bit codes,
	// 172 bits in all. Apart, it writes 32 bits of allocations, one channel's 20 bits of selection
	// and scalefactors, and the same codes, 160 bits in all.
	assert.Len(t, shared, (24+40+108+7)/8)
	assert.Len(t, apart, (32+20+108)/8)
}

// TestRandomLayer2Audio checks that random fields keep to the table, and that the parts a
// selection joins share their scalefactor, over enough subbands to draw every selection.
func TestRandomLayer2Audio(t *testing.T) {
	t.Parallel()

	table := make([]Layer2Subband, 0, 16)
	for range 8 {
		table = append(table, Layer2Subband{Steps: []int{3, 5, 9}, Width: 2})
		table = append(table, Layer2Subband{Steps: []int{3, 7, 15, 31, 63, 127, 255}, Width: 3})
	}

	fields := RandomLayer2Audio(rand.New(rand.NewPCG(1, 2)), table)
	drawn := make(map[uint32]bool)

	for ch := range fields.Allocation {
		for sb, subband := range table {
			allocation := fields.Allocation[ch][sb]
			factors := fields.Scalefactors[ch][sb]

			assert.Less(t, allocation, uint32(1)<<subband.Width)
			assert.Less(t, fields.Selections[ch][sb], uint32(4))

			for _, factor := range factors {
				assert.LessOrEqual(t, factor, uint32(62))
			}

			drawn[fields.Selections[ch][sb]] = true

			switch fields.Selections[ch][sb] {
			case 1:
				assert.Equal(t, factors[0], factors[1])
			case 2:
				assert.Equal(t, [Layer2Parts]uint32{factors[0], factors[0], factors[0]}, factors)
			case 3:
				assert.Equal(t, factors[1], factors[2])
			default:
			}

			for set := range Layer2Sets {
				if allocation > 0 {
					assert.Less(t, fields.Samples[set][ch][sb], uint32(subband.Steps[allocation-1]))
				}
			}
		}
	}

	assert.Len(t, drawn, 4, "selections drawn")
}
