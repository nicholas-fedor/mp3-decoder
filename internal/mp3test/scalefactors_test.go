// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMPEG1ScalefactorLengths checks the number of bits written at each scalefac_compress against
// the published part2 lengths: 11 slen1 and 10 slen2 for long blocks, 18 of each for short blocks,
// and 17 slen1 and 18 slen2 for mixed blocks.
func TestMPEG1ScalefactorLengths(t *testing.T) {
	t.Parallel()

	for compress := range uint32(16) {
		first, second := slen1[compress], slen2[compress]

		tests := []struct {
			name    string
			granule Granule
			reuse   [4]bool
			want    int
		}{
			{"long", Granule{ScalefacCompress: compress}, [4]bool{}, 11*first + 10*second},
			{"long reusing groups 0 and 2", Granule{ScalefacCompress: compress}, [4]bool{true, false, true, false}, 5*first + 5*second},
			{"start block", Granule{ScalefacCompress: compress, WindowSwitching: true, BlockType: 1}, [4]bool{}, 11*first + 10*second},
			{"short", Granule{ScalefacCompress: compress, WindowSwitching: true, BlockType: 2}, [4]bool{}, 18*first + 18*second},
			{"mixed", Granule{ScalefacCompress: compress, WindowSwitching: true, BlockType: 2, MixedBlock: true}, [4]bool{}, 17*first + 18*second},
		}

		for _, tt := range tests {
			var w BitWriter

			w.MPEG1Scalefactors(&tt.granule, tt.reuse, &Scalefactors{})

			assert.Equal(t, tt.want, w.Len(), "%s at scalefac_compress %d", tt.name, compress)
		}
	}
}

// TestMPEG1ScalefactorOrder checks the bits of long blocks at slen1 3 and slen2 0, and of mixed
// blocks, whose long bands come before their short bands.
func TestMPEG1ScalefactorOrder(t *testing.T) {
	t.Parallel()

	var long BitWriter

	factors := Scalefactors{}
	factors.Long[0], factors.Long[10] = 0b101, 0b111

	long.MPEG1Scalefactors(&Granule{ScalefacCompress: 4}, [4]bool{}, &factors)

	assert.Equal(t, []byte{0b1010_0000, 0, 0, 0b0000_0011, 0b1000_0000}, long.Bytes())

	var mixed BitWriter

	factors = Scalefactors{}
	factors.Long[7], factors.Short[3][2] = 1, 1

	mixed.MPEG1Scalefactors(&Granule{ScalefacCompress: 5, WindowSwitching: true, BlockType: 2, MixedBlock: true}, [4]bool{}, &factors)

	assert.Equal(t, []byte{0b0000_0001, 0b0010_0000, 0, 0, 0}, mixed.Bytes())
	assert.Equal(t, 35, mixed.Len())
}
