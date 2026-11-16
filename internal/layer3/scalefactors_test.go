// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// mpeg1Head is an MPEG-1 Layer III stereo header, which takes the MPEG-1 scalefactors.
var mpeg1Head = frame.Header{SampleRate: 44100, Version: frame.MPEG1, Layer: frame.Layer3, Mode: frame.Stereo}

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
			factors.Read(&stream, mpeg1Head, &side, 0, 0)

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

	factors.shift = mpeg1MixedShift

	assert.Equal(t, 7, factors.Long(7))
	assert.Equal(t, 8, factors.Short(3, 0))
	assert.Equal(t, 37, factors.Short(12, 2))

	factors.shift = lsfMixedShift

	assert.Equal(t, 5, factors.Long(5))
	assert.Equal(t, 6, factors.Short(3, 0))
	assert.Equal(t, 35, factors.Short(12, 2))
}

// TestLSFCounts checks that every row of the lower-rate partition counts sends long bands 0 to 20,
// each window of short bands 0 to 11, or long bands 0 to 5 and each window of short bands 3 to 11,
// and that the long bands of mixed blocks fit in their first partition.
func TestLSFCounts(t *testing.T) {
	t.Parallel()

	for table, row := range lsfCounts {
		for blocks, want := range [3]int{21, 36, 6 + 27} {
			sum := 0
			for _, count := range row[blocks] {
				sum += count
			}

			assert.Equal(t, want, sum, "table %d blocks %d", table, blocks)
		}

		assert.GreaterOrEqual(t, row[2][0], 6, "table %d", table)
	}
}

// TestReadLSF checks the bits read and preflag against worked examples of each range of
// scalefac_compress.
//
// Preflag comes from values 500 and up, and never in the right channel of an intensity stereo
// frame, whose value is halved.
func TestReadLSF(t *testing.T) {
	t.Parallel()

	plain := frame.Header{SampleRate: 22050, Version: frame.MPEG2, Layer: frame.Layer3, Mode: frame.JointStereo}
	intensity := plain
	intensity.Extension = intensityStereo

	short := GranuleInfo{BlockType: ShortBlocks}
	mixed := GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}

	tests := []struct {
		name     string
		head     frame.Header
		info     GranuleInfo
		compress int
		ch       int
		bits     int
		preflag  bool
	}{
		{"399: widths 4 4 3 3, counts 6 5 5 5", plain, GranuleInfo{}, 399, 0, 74, false},
		{"399 short: counts 9 9 9 9", plain, short, 399, 1, 126, false},
		{"399 mixed: counts 6 9 9 9", plain, mixed, 399, 0, 114, false},
		{"499: widths 4 4 3 0, counts 6 5 7 3", plain, GranuleInfo{}, 499, 0, 65, false},
		{"500: preflag", plain, GranuleInfo{}, 500, 0, 0, true},
		{"511: widths 3 2 0 0, counts 11 10", intensity, GranuleInfo{}, 511, 0, 53, true},
		{"intensity 359: widths 4 5 5 0, counts 7 7 7", intensity, GranuleInfo{}, 359, 1, 98, false},
		{"intensity 487: widths 3 3 3 0, counts 6 6 6 3", intensity, GranuleInfo{}, 487, 1, 54, false},
		{"intensity 511 short: counts 15 12 9", intensity, short, 511, 1, 69, false},
	}

	data := make([]byte, 64)

	for _, tt := range tests {
		var (
			stream  bits.Reader
			side    SideInfo
			factors Scalefactors
		)

		tt.info.ScalefacCompress = tt.compress
		side.Granules[0][tt.ch] = tt.info
		stream.Reset(data, 0)
		factors.Read(&stream, tt.head, &side, 0, tt.ch)

		assert.Equal(t, tt.bits, stream.Pos(), tt.name)
		assert.Equal(t, tt.preflag, factors.preflag, tt.name)
	}
}

// TestWidest checks the largest value each scalefactor's width holds, at the lower sampling
// frequencies and in MPEG-1, with 0 for the bands no partition sends.
func TestWidest(t *testing.T) {
	t.Parallel()

	data := make([]byte, 64)
	lsf := frame.Header{SampleRate: 22050, Version: frame.MPEG2, Layer: frame.Layer3, Mode: frame.Stereo}

	var (
		stream  bits.Reader
		side    SideInfo
		factors Scalefactors
	)

	// 399 gives widths 4, 4, 3 and 3 over long bands 0 to 5, 6 to 10, 11 to 15 and 16 to 20.
	side.Granules[0][0] = GranuleInfo{ScalefacCompress: 399}
	stream.Reset(data, 0)
	factors.Read(&stream, lsf, &side, 0, 0)

	assert.Equal(t, 15, factors.widestLong(5))
	assert.Equal(t, 15, factors.widestLong(6))
	assert.Equal(t, 7, factors.widestLong(11))
	assert.Equal(t, 7, factors.widestLong(20))
	assert.Zero(t, factors.widestLong(21))

	// MPEG-1 scalefac_compress 14 gives slen1 4 for short bands 0 to 5 and slen2 2 for 6 to 11.
	side.Granules[0][0] = GranuleInfo{ScalefacCompress: 14, BlockType: ShortBlocks}
	stream.Reset(data, 0)
	factors.Read(&stream, mpeg1Head, &side, 0, 0)

	assert.Equal(t, 15, factors.widestShort(5, 2))
	assert.Equal(t, 3, factors.widestShort(6, 0))
	assert.Zero(t, factors.widestShort(12, 1))
}
