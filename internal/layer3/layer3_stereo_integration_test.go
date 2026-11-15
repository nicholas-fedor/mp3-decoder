// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestStereoIntensityFrame decodes a joint stereo granule written as bits, with mid/side and
// intensity stereo, and checks each long band's lines at 44.1 kHz.
//
// The right channel holds values only in long bands 0 to 7, so bands 8 to 21 lie above the
// intensity bound. Its scalefactors give each band's position, band 21 takes that of band 20, and
// positions of 7 or more code none.
func TestStereoIntensityFrame(t *testing.T) {
	t.Parallel()

	edges := [23]int{0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 52, 62, 74, 90, 110, 134, 162, 196, 238, 288, 342, 418, 576}
	head := frame.Header{SampleRate: 44100, Version: frame.MPEG1, Layer: frame.Layer3, Mode: frame.JointStereo, Extension: 3}
	rng := rand.New(rand.NewPCG(17, 18))

	var (
		side    layer3.SideInfo
		writers [2]mp3test.BitWriter
		factors [2]layer3.Scalefactors
		lines   [2][576]float64
	)

	side.Granules[0][0] = layer3.GranuleInfo{BigValues: 288, TableSelect: [3]int{15, 15, 15}, GlobalGain: 210}
	side.Granules[0][1] = layer3.GranuleInfo{BigValues: 18, TableSelect: [3]int{1, 1, 1}, GlobalGain: 210, ScalefacCompress: 15}

	positions := randomScalefactors(rng, slens[15])
	writers[1].MPEG1Scalefactors(&mp3test.Granule{ScalefacCompress: 15}, [4]bool{}, &positions)

	signs := make([]int, 36)
	for at := range signs {
		signs[at] = 1 - 2*rng.IntN(2)
	}

	writers[0].BigValues(15, randomPairs(rng, 15, 576))
	writers[1].BigValues(1, signs)

	for ch := range writers {
		var (
			stream   bits.Reader
			spectrum layer3.Spectrum
		)

		info := &side.Granules[0][ch]

		stream.Reset(writers[ch].Bytes(), 0)
		factors[ch].Read(&stream, head, &side, 0, ch)
		spectrum.Read(&stream, head, info, writers[ch].Len())
		spectrum.Requantize(&lines[ch], info, &factors[ch])
	}

	coded := lines

	layer3.Stereo(head, &lines[0], &lines[1], &side.Granules[0][1], &factors[1])

	for band := range 22 {
		position := int(positions.Long[min(band, 20)])

		for line := edges[band]; line < edges[band+1]; line++ {
			mid, other := coded[0][line], coded[1][line]
			wantLeft, wantRight := (mid+other)/math.Sqrt2, (mid-other)/math.Sqrt2

			if band >= 8 && position < 7 {
				k := math.Tan(float64(position) * math.Pi / 12)
				wantLeft, wantRight = mid*k/(1+k), mid/(1+k)
			}

			assert.InDelta(t, wantLeft, lines[0][line], 1e-12, "band %d line %d left", band, line)
			assert.InDelta(t, wantRight, lines[1][line], 1e-12, "band %d line %d right", band, line)
		}
	}
}

// TestStereoEncodedFiles restores and reorders every granule of a joint stereo file LAME encoded.
//
// LAME codes some granules with mid/side stereo, and mid/side stereo must keep each line's energy.
// Reordering must keep each channel's lines.
func TestStereoEncodedFiles(t *testing.T) {
	t.Parallel()

	var (
		side      layer3.SideInfo
		reservoir layer3.Reservoir
		stream    bits.Reader
		factors   [2]layer3.Scalefactors
		spectrum  layer3.Spectrum
		lines     [2][576]float64
	)

	midSide, short := 0, 0

	for at, data := range encodedFrames(t, "mpeg1-joint.mp3") {
		head, _ := frame.Parse(data)
		require.NoError(t, side.Parse(head, data))
		require.Equal(t, frame.JointStereo, head.Mode)

		main, _ := reservoir.MainData(side.MainDataBegin, data[head.DataAt():])
		start := 0

		for gr := range 2 {
			for ch := range 2 {
				info := &side.Granules[gr][ch]

				stream.Reset(main, start)
				factors[ch].Read(&stream, head, &side, gr, ch)
				spectrum.Read(&stream, head, info, start+info.Part23Length)
				spectrum.Requantize(&lines[ch], info, &factors[ch])

				start += info.Part23Length
			}

			coded := lines

			layer3.Stereo(head, &lines[0], &lines[1], &side.Granules[gr][1], &factors[1])

			if head.Extension == 2 {
				midSide++

				for line := range 576 {
					energy := coded[0][line]*coded[0][line] + coded[1][line]*coded[1][line]
					got := lines[0][line]*lines[0][line] + lines[1][line]*lines[1][line]
					require.InDelta(t, energy, got, 1e-9*max(energy, 1), "frame %d granule %d line %d", at, gr, line)
				}
			}

			for ch := range 2 {
				info := &side.Granules[gr][ch]
				if info.BlockType == layer3.ShortBlocks {
					short++
				}

				before := slices.Sorted(slices.Values(lines[ch][:]))

				layer3.Reorder(head, &lines[ch], info)
				require.Equal(t, before, slices.Sorted(slices.Values(lines[ch][:])), "frame %d granule %d", at, gr)
			}
		}
	}

	assert.Positive(t, midSide, "mid/side granules")
	assert.Positive(t, short, "short block granules")
}
