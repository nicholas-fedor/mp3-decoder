// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

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

// lsfLength is a worked example of the bits lower-rate scalefactors take.
type lsfLength struct {
	// name names the example.
	name string
	// granule is the granule's side information.
	granule Granule
	// want is the number of bits.
	want int
}

// lsfLengths holds worked examples at the edges of each range of scalefac_compress, with the
// widths and partition counts each range gives.
var lsfLengths = []lsfLength{
	{"0", Granule{ScalefacCompress: 0}, 0},
	{"399: widths 4 4 3 3, counts 6 5 5 5", Granule{ScalefacCompress: 399}, 74},
	{"399 short: counts 9 9 9 9", Granule{ScalefacCompress: 399, WindowSwitching: true, BlockType: 2}, 126},
	{"399 mixed: counts 6 9 9 9", Granule{ScalefacCompress: 399, WindowSwitching: true, BlockType: 2, MixedBlock: true}, 114},
	{"400", Granule{ScalefacCompress: 400}, 0},
	{"499: widths 4 4 3 0, counts 6 5 7 3", Granule{ScalefacCompress: 499}, 65},
	{"511: widths 3 2 0 0, counts 11 10", Granule{ScalefacCompress: 511}, 53},
	{"511 mixed: counts 15 18", Granule{ScalefacCompress: 511, WindowSwitching: true, BlockType: 2, MixedBlock: true}, 81},
}

// intensityLengths holds worked examples for the right channel of an intensity stereo frame,
// whose scalefac_compress halved gives the widths and counts.
var intensityLengths = []lsfLength{
	{"359: widths 4 5 5 0, counts 7 7 7", Granule{ScalefacCompress: 359}, 98},
	{"487: widths 3 3 3 0, counts 6 6 6 3", Granule{ScalefacCompress: 487}, 54},
	{"511: widths 3 2 0 0, counts 8 8 5", Granule{ScalefacCompress: 511}, 40},
	{"511 short: counts 15 12 9", Granule{ScalefacCompress: 511, WindowSwitching: true, BlockType: 2}, 69},
}

// TestLSFScalefactorLengths checks the bits written against worked examples of each range of
// scalefac_compress, for every channel and for the right channel of an intensity stereo frame.
func TestLSFScalefactorLengths(t *testing.T) {
	t.Parallel()

	for _, tt := range lsfLengths {
		var w BitWriter

		w.LSFScalefactors(&tt.granule, &Scalefactors{})
		assert.Equal(t, tt.want, w.Len(), tt.name)
	}

	for _, tt := range intensityLengths {
		var w BitWriter

		w.IntensityScalefactors(&tt.granule, &Scalefactors{})
		assert.Equal(t, tt.want, w.Len(), tt.name)
	}
}

// TestLSFScalefactorsCut checks that each scalefactor keeps only the bits its width holds, and that
// long bands past the 6 that mixed blocks write stay as they were.
func TestLSFScalefactorsCut(t *testing.T) {
	t.Parallel()

	var (
		w       BitWriter
		factors Scalefactors
	)

	for band := range factors.Long {
		factors.Long[band] = 0xff
	}

	for band := range factors.Short {
		factors.Short[band] = [3]uint32{0xff, 0xff, 0xff}
	}

	// 399 gives widths 4, 4, 3 and 3 over counts 6, 9, 9 and 9 for mixed blocks.
	w.LSFScalefactors(&Granule{ScalefacCompress: 399, WindowSwitching: true, BlockType: 2, MixedBlock: true}, &factors)

	assert.Equal(t, [21]uint32{
		15, 15, 15, 15, 15, 15, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff,
	}, factors.Long)
	assert.Equal(t, [3]uint32{0xff, 0xff, 0xff}, factors.Short[2])
	assert.Equal(t, [3]uint32{15, 15, 15}, factors.Short[5])
	assert.Equal(t, [3]uint32{7, 7, 7}, factors.Short[8])
	assert.Equal(t, [3]uint32{7, 7, 7}, factors.Short[11])
}
