// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// TestReadMPEG1Lengths checks the number of bits read at each scalefac_compress against the
// published part2 lengths: 11 slen1 and 10 slen2 for long blocks, 18 of each for short blocks, and
// 17 slen1 and 18 slen2 for mixed blocks.
func TestReadMPEG1Lengths(t *testing.T) {
	t.Parallel()

	data := make([]byte, 64)

	for compress, slen := range slens {
		tests := []struct {
			name string
			info GranuleInfo
			want int
		}{
			{"long", GranuleInfo{ScalefacCompress: compress}, 11*slen[0] + 10*slen[1]},
			{"stop block", GranuleInfo{ScalefacCompress: compress, BlockType: StopBlock}, 11*slen[0] + 10*slen[1]},
			{"short", GranuleInfo{ScalefacCompress: compress, BlockType: ShortBlocks}, 18*slen[0] + 18*slen[1]},
			{"mixed", GranuleInfo{ScalefacCompress: compress, BlockType: ShortBlocks, MixedBlock: true}, 17*slen[0] + 18*slen[1]},
		}

		for _, tt := range tests {
			var (
				stream  bits.Reader
				side    SideInfo
				factors Scalefactors
			)

			side.Granules[0][0] = tt.info
			stream.Reset(data, 0)
			factors.ReadMPEG1(&stream, &side, 0, 0)

			assert.Equal(t, tt.want, stream.Pos(), "%s at scalefac_compress %d", tt.name, compress)
		}
	}
}

// TestShortPositions checks that mixed blocks' short band 3 sits right after their 8 long bands,
// and that short blocks' band 3 sits after the 9 values of bands 0 to 2.
func TestShortPositions(t *testing.T) {
	t.Parallel()

	var factors Scalefactors

	for at := range factors.values {
		factors.values[at] = uint8(at)
	}

	assert.Equal(t, 9, factors.Short(3, 0))
	assert.Equal(t, 38, factors.Short(12, 2))

	factors.mixed = true

	assert.Equal(t, 7, factors.Long(7))
	assert.Equal(t, 8, factors.Short(3, 0))
	assert.Equal(t, 37, factors.Short(12, 2))
}
