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
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestSpectrumRoundTrip writes random values in each region's table and count1 quadruples for
// granules of every kind of block, and reads them back to the last bit.
//
// Each case gives its region boundaries from the published band edges.
func TestSpectrumRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		info   layer3.GranuleInfo
		rate   int
		starts [2]int
		quads  int
	}{
		{"long", layer3.GranuleInfo{BigValues: 100, TableSelect: [3]int{1, 13, 24}, Region0Count: 3, Region1Count: 2}, 44100, [2]int{16, 30}, 10},
		{"long in region0", layer3.GranuleInfo{BigValues: 50, TableSelect: [3]int{15, 0, 0}, Region0Count: 15, Region1Count: 7}, 44100, [2]int{162, 576}, 0},
		{"unused tables", layer3.GranuleInfo{BigValues: 40, TableSelect: [3]int{4, 14, 5}, Count1TableSelect: 1}, 48000, [2]int{4, 8}, 3},
		{"short", layer3.GranuleInfo{BigValues: 150, TableSelect: [3]int{7, 16}, BlockType: layer3.ShortBlocks, Count1TableSelect: 1}, 32000, [2]int{36, 576}, 20},
		{"mixed", layer3.GranuleInfo{BigValues: 288, TableSelect: [3]int{2, 31}, BlockType: layer3.ShortBlocks, MixedBlock: true}, 44100, [2]int{36, 576}, 0},
		{"start", layer3.GranuleInfo{BigValues: 80, TableSelect: [3]int{0, 9}, BlockType: layer3.StartBlock}, 22050, [2]int{54, 576}, 5},
		{"stop at 8 kHz", layer3.GranuleInfo{BigValues: 70, TableSelect: [3]int{11, 12}, BlockType: layer3.StopBlock}, 8000, [2]int{108, 576}, 7},
	}

	for seed, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var w mp3test.BitWriter

			rng := rand.New(rand.NewPCG(9, uint64(seed)))

			lines := 2 * tt.info.BigValues
			starts := []int{0, min(tt.starts[0], lines), min(tt.starts[1], lines), lines}
			want := make([]int, 576)

			for region, table := range tt.info.TableSelect {
				values := randomPairs(rng, table, starts[region+1]-starts[region])
				copy(want[starts[region]:], values)
				w.BigValues(table, values)
			}

			quads := make([]int, 4*tt.quads)
			for at := range quads {
				quads[at] = rng.IntN(3) - 1
			}

			copy(want[lines:], quads)
			w.Count1(tt.info.Count1TableSelect, quads)

			got := readSpectrum(t, w.Bytes(), tt.rate, &tt.info, w.Len())

			assert.Equal(t, want, got.Values[:])
			assert.Equal(t, lines+len(quads), got.Decoded)
		})
	}
}

// TestSpectrumEnds checks where decoding stops: at a big_values pair that starts before the end,
// at a count1 quadruple that runs past it, and at 576 values with bits to spare.
func TestSpectrumEnds(t *testing.T) {
	t.Parallel()

	ones := []int{1, -1, 1, -1, 1, -1, 1, -1}

	t.Run("big_values", func(t *testing.T) {
		t.Parallel()

		var w mp3test.BitWriter

		info := layer3.GranuleInfo{BigValues: 4, TableSelect: [3]int{1, 1, 1}}

		w.BigValues(1, ones[:4])
		third := w.Len()
		w.BigValues(1, ones[4:])

		got := readSpectrum(t, w.Bytes(), 44100, &info, third+1)

		assert.Equal(t, ones[:6], got.Values[:6])
		assert.Equal(t, 6, got.Decoded)
		assert.Zero(t, got.Values[6])
	})

	t.Run("count1", func(t *testing.T) {
		t.Parallel()

		var w mp3test.BitWriter

		info := layer3.GranuleInfo{}

		w.Count1(0, ones[:4])
		w.Count1(0, ones[4:])

		got := readSpectrum(t, w.Bytes(), 44100, &info, w.Len()-1)

		assert.Equal(t, ones[:4], got.Values[:4])
		assert.Equal(t, 4, got.Decoded)
		assert.Zero(t, got.Values[4])
	})

	t.Run("576 values", func(t *testing.T) {
		t.Parallel()

		var w mp3test.BitWriter

		info := layer3.GranuleInfo{BigValues: 280, TableSelect: [3]int{1, 1, 1}, Count1TableSelect: 1}
		quads := slices.Concat(ones, ones, ones)

		w.BigValues(1, make([]int, 560))
		w.Count1(1, quads)

		got := readSpectrum(t, w.Bytes(), 44100, &info, w.Len())

		assert.Equal(t, 576, got.Decoded)
		assert.Equal(t, quads[:16], got.Values[560:])
	})
}

// TestSpectrumEncodedFiles decodes the scalefactors and spectral values of every granule of MPEG-1
// files LAME encoded, and requantizes them.
//
// Each granule must decode all its big_values pairs and end on its last bit, unless its values fill
// the granule first, leaving no room for another quadruple. Each spectral value must be finite, and
// share its quantized value's sign and whether it is 0.
func TestSpectrumEncodedFiles(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				side      layer3.SideInfo
				reservoir layer3.Reservoir
				stream    bits.Reader
				factors   [2]layer3.Scalefactors
				spectrum  layer3.Spectrum
				lines     [576]float64
			)

			for at, data := range encodedFrames(t, name) {
				head, _ := frame.Parse(data)
				require.NoError(t, side.Parse(head, data))

				main, _ := reservoir.MainData(side.MainDataBegin, data[head.DataAt():])
				start := 0

				for gr := range 2 {
					for ch := range head.Channels() {
						info := &side.Granules[gr][ch]
						end := start + info.Part23Length

						stream.Reset(main, start)
						factors[ch].Read(&stream, head, &side, gr, ch)
						spectrum.Read(&stream, head, info, end)

						assert.GreaterOrEqual(t, spectrum.Decoded, 2*info.BigValues, "frame %d", at)

						spectrum.Requantize(&lines, info, &factors[ch])
						assertRequantized(t, &spectrum, &lines, at)

						if spectrum.Decoded+4 <= 576 {
							assert.Equal(t, end, stream.Pos(), "frame %d granule %d channel %d", at, gr, ch)
						}

						start = end
					}
				}
			}
		})
	}
}

// assertRequantized checks that each spectral value is finite and shares its quantized value's sign
// and whether it is 0.
func assertRequantized(t *testing.T, spectrum *layer3.Spectrum, lines *[576]float64, frame int) {
	t.Helper()

	for line, value := range lines {
		quantized := spectrum.Values[line]

		require.False(t, math.IsInf(value, 0) || math.IsNaN(value), "frame %d line %d", frame, line)
		require.Equal(t, quantized == 0, value == 0, "frame %d line %d", frame, line)
		require.Equal(t, quantized < 0, value < 0, "frame %d line %d", frame, line)
	}
}

// readSpectrum decodes the spectral values of a granule from bits that hold only its Huffman data,
// into values that start as 99 so that any value it leaves shows.
func readSpectrum(t *testing.T, data []byte, rate int, info *layer3.GranuleInfo, end int) *layer3.Spectrum {
	t.Helper()

	var (
		stream   bits.Reader
		spectrum layer3.Spectrum
	)

	for at := range spectrum.Values {
		spectrum.Values[at] = 99
	}

	stream.Reset(data, 0)
	spectrum.Read(&stream, frame.Header{SampleRate: rate, Version: frame.MPEG1, Layer: frame.Layer3}, info, end)

	return &spectrum
}

// randomPairs returns random values within the range of a big_values table, two for each pair.
func randomPairs(rng *rand.Rand, table, count int) []int {
	values := make([]int, count)

	codes, ok := codebook.BigValue(table)
	if !ok {
		return values
	}

	limit := codes.Width()
	if codes.Linbits() > 0 {
		limit = 15 + 1<<codes.Linbits()
	}

	for at := range values {
		values[at] = rng.IntN(2*limit-1) - (limit - 1)
	}

	return values
}
