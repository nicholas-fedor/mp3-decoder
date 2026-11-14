// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"bytes"
	"fmt"
	"go/format"
	"math"
	"math/big"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// lineBand is the band and window a line of a granule belongs to.
type lineBand struct {
	// band is the long or short band.
	band int
	// window is the short window, or 0 for a long band.
	window int
	// short reports whether the band is a short band.
	short bool
}

// Stereo test constants.
const (
	// stereoFile is the file the intensity ratios are generated into.
	stereoFile = "stereo_gen.go"
	// stereoTolerance is the largest difference the stereo oracle may show for lines between -1 and
	// 1.
	stereoTolerance = 1e-15
)

// mpeg1Rates holds the MPEG-1 sampling frequencies.
var mpeg1Rates = []int{44100, 48000, 32000}

// TestStereoTables checks that the generated intensity ratios hold k/(1+k) for k = tan(p π/12),
// rounded to the nearest float64, or rewrites them with -update.
//
// The closed forms come from tan(π/12) = 2-√3 and tan(π/6) = 1/√3. The ratios must also agree with
// math.Tan, and each position's two ratios must sum to 1.
func TestStereoTables(t *testing.T) {
	var ratios [noPosition]float64

	for position := range ratios {
		ratios[position] = intensityClosed(position)
	}

	if *update {
		require.NoError(t, os.WriteFile(stereoFile, stereoSource(t, &ratios), 0o644))

		return
	}

	t.Parallel()

	assert.Equal(t, ratios, intensityRatios)

	half := new(big.Float).SetPrec(precision).SetInt64(2)
	half.Sqrt(half).Quo(half, big.NewFloat(2))

	root, _ := half.Float64()
	assert.InDelta(t, invSqrt2, root, 0)

	for position, ratio := range intensityRatios {
		k := math.Tan(float64(position) * math.Pi / 12)
		assert.InDelta(t, k/(1+k), ratio, stereoTolerance, "position %d", position)
		assert.InDelta(t, 1, ratio+intensityRatios[noPosition-1-position], 0x1p-52, "position %d", position)
	}
}

// TestStereo checks joint stereo granules of every kind of block at each MPEG-1 sampling frequency
// against the formulas, line by line, in every mode.
//
// The right channel holds values up to a random band in each window, with silent bands below it,
// and the positions include those that code none.
func TestStereo(t *testing.T) {
	t.Parallel()

	kinds := []GranuleInfo{
		{},
		{BlockType: StartBlock},
		{BlockType: StopBlock},
		{BlockType: ShortBlocks},
		{BlockType: ShortBlocks, MixedBlock: true},
	}

	rng := rand.New(rand.NewPCG(13, 14))

	for _, widths := range publishedWidths[:len(mpeg1Rates)] {
		for _, kind := range kinds {
			for extension := range uint8(4) {
				for range 20 {
					head := frame.Header{
						SampleRate: widths.rate, Version: frame.MPEG1, Layer: frame.Layer3,
						Mode: frame.JointStereo, Extension: extension,
					}
					places := linePlaces(&kind, widths.long, widths.short)
					left, right := randomStereo(rng, &places)
					factors := randomPositions(rng, kind.BlockType == ShortBlocks && kind.MixedBlock)

					wantLeft, wantRight := stereoOracle(head, &left, &right, &places, factors)

					Stereo(head, &left, &right, &kind, factors)

					name := fmt.Sprintf("%d Hz %+v extension %d", widths.rate, kind, extension)
					assertLines(t, &wantLeft, &left, name+" left")
					assertLines(t, &wantRight, &right, name+" right")
				}
			}
		}
	}
}

// TestStereoBands checks one long block granule that uses mid/side and intensity stereo together.
//
// The bands below the intensity bound and a band that codes no position take mid/side stereo, and
// the bands above it take intensity stereo alone. The highest band takes the position of band 20.
func TestStereoBands(t *testing.T) {
	t.Parallel()

	head := frame.Header{SampleRate: 44100, Version: frame.MPEG1, Layer: frame.Layer3, Mode: frame.JointStereo, Extension: 3}

	var left, right [granuleLines]float64

	for line := range left {
		left[line] = 1
	}

	right[61] = 0.5

	factors := &Scalefactors{}
	factors.values[11] = noPosition
	factors.values[15] = 3
	factors.values[20] = 6

	Stereo(head, &left, &right, &GranuleInfo{}, factors)

	assert.InDelta(t, 1.5*invSqrt2, left[61], stereoTolerance, "band 10, below the bound")
	assert.InDelta(t, 0.5*invSqrt2, right[61], stereoTolerance, "band 10, below the bound")
	assert.InDelta(t, invSqrt2, left[62], stereoTolerance, "band 11, no position")
	assert.InDelta(t, invSqrt2, right[62], stereoTolerance, "band 11, no position")
	assert.InDelta(t, 0.5, left[134], stereoTolerance, "band 15, position 3")
	assert.InDelta(t, 0.5, right[134], stereoTolerance, "band 15, position 3")
	assert.InDelta(t, 1, left[575], 0, "band 21, position 6 of band 20")
	assert.Zero(t, right[575], "band 21, position 6 of band 20")
}

// TestStereoModes checks that frames that are not joint stereo, and joint stereo frames with
// neither mode, keep their lines, and that mid/side stereo keeps each line's energy and undoes
// itself.
func TestStereoModes(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(15, 16))
	places := linePlaces(&GranuleInfo{}, publishedWidths[0].long, publishedWidths[0].short)
	left, right := randomStereo(rng, &places)

	for _, head := range []frame.Header{
		{SampleRate: 44100, Mode: frame.Stereo, Extension: 3},
		{SampleRate: 44100, Mode: frame.DualChannel, Extension: 3},
		{SampleRate: 44100, Mode: frame.JointStereo},
	} {
		gotLeft, gotRight := left, right
		Stereo(head, &gotLeft, &gotRight, &GranuleInfo{}, &Scalefactors{})
		assert.Equal(t, left, gotLeft, "%+v", head)
		assert.Equal(t, right, gotRight, "%+v", head)
	}

	head := frame.Header{SampleRate: 44100, Mode: frame.JointStereo, Extension: midSideStereo}
	gotLeft, gotRight := left, right

	Stereo(head, &gotLeft, &gotRight, &GranuleInfo{}, &Scalefactors{})

	for line := range left {
		energy := left[line]*left[line] + right[line]*right[line]
		assert.InDelta(t, energy, gotLeft[line]*gotLeft[line]+gotRight[line]*gotRight[line], stereoTolerance, "line %d", line)
	}

	Stereo(head, &gotLeft, &gotRight, &GranuleInfo{}, &Scalefactors{})
	assertLines(t, &left, &gotLeft, "left after two passes")
	assertLines(t, &right, &gotRight, "right after two passes")
}

// linePlaces maps each line of a granule to its band and window, from band widths.
//
// Mixed blocks hold long bands up to where short band 3 starts.
func linePlaces(info *GranuleInfo, long [longBands]int, short [shortBands]int) [granuleLines]lineBand {
	var places [granuleLines]lineBand

	line := 0
	mixedEnd := 3 * (short[0] + short[1] + short[2])

	if info.BlockType != ShortBlocks || info.MixedBlock {
		for band, width := range long {
			if info.BlockType == ShortBlocks && line >= mixedEnd {
				break
			}

			for range width {
				places[line] = lineBand{band: band}
				line++
			}
		}
	}

	if info.BlockType != ShortBlocks {
		return places
	}

	start := 0

	for band, width := range short {
		start += 3 * width
		if start <= line {
			continue
		}

		for window := range 3 {
			for range width {
				places[line] = lineBand{band: band, window: window, short: true}
				line++
			}
		}
	}

	return places
}

// randomStereo returns random left and right lines between -1 and 1.
//
// The right channel holds values up to a random band in each window, or in none. Below it each band
// is silent one time in four.
func randomStereo(rng *rand.Rand, places *[granuleLines]lineBand) ([granuleLines]float64, [granuleLines]float64) {
	var (
		left, right [granuleLines]float64
		tops        [4]int
		silent      [4][longBands]bool
	)

	for at := range tops {
		tops[at] = rng.IntN(longBands+1) - 1

		for band := range silent[at] {
			silent[at][band] = rng.IntN(4) == 0
		}
	}

	for line, place := range places {
		left[line] = 2*rng.Float64() - 1

		kind := place.window
		if place.short {
			kind++
		}

		if place.band <= tops[kind] && !silent[kind][place.band] {
			right[line] = 2*rng.Float64() - 1
		}
	}

	return left, right
}

// randomPositions returns random scalefactors that hold intensity positions from 0 to 8, so that
// positions 7 and 8 code none.
func randomPositions(rng *rand.Rand, mixed bool) *Scalefactors {
	factors := &Scalefactors{mixed: mixed}

	for at := range shortSent {
		factors.values[at] = uint8(rng.IntN(noPosition + 2))
	}

	return factors
}

// stereoOracle restores joint stereo lines by the formulas.
//
// It finds the intensity bound of the long bands and of each window first, from the highest band
// in which the right channel holds a value, then restores each line on its own.
func stereoOracle(
	head frame.Header, left, right *[granuleLines]float64, places *[granuleLines]lineBand, factors *Scalefactors,
) ([granuleLines]float64, [granuleLines]float64) {
	tops := [4]int{-1, -1, -1, -1}

	for line, place := range places {
		kind := place.window
		if place.short {
			kind++
		}

		if right[line] != 0 {
			tops[kind] = max(tops[kind], place.band)
		}
	}

	shortsSilent := tops[1] < 0 && tops[2] < 0 && tops[3] < 0
	outLeft, outRight := *left, *right

	for line, place := range places {
		coded := place.band > tops[0] && shortsSilent
		position := factors.Long(min(place.band, 20))

		if place.short {
			coded = place.band > tops[place.window+1]
			position = factors.Short(min(place.band, 11), place.window)
		}

		switch value := left[line]; {
		case head.Extension&1 != 0 && coded && position < 7:
			k := math.Tan(float64(position) * math.Pi / 12)
			outLeft[line], outRight[line] = value*k/(1+k), value/(1+k)

			if position == 6 {
				outLeft[line], outRight[line] = value, 0
			}
		case head.Extension&2 != 0:
			outLeft[line] = (value + right[line]) / math.Sqrt2
			outRight[line] = (value - right[line]) / math.Sqrt2
		default:
		}
	}

	return outLeft, outRight
}

// assertLines checks lines against the lines an oracle gives.
func assertLines(t *testing.T, want, got *[granuleLines]float64, name string) {
	t.Helper()

	for line := range want {
		if math.Abs(want[line]-got[line]) > stereoTolerance {
			require.InDelta(t, want[line], got[line], stereoTolerance, "%s line %d", name, line)
		}
	}
}

// intensityClosed returns k/(1+k) for k = tan(p π/12), rounded to the nearest float64, from its
// closed form in √3.
func intensityClosed(position int) float64 {
	root := new(big.Float).SetPrec(precision).SetInt64(3)
	root.Sqrt(root)

	value := func(a, b, c int64) *big.Float {
		sum := new(big.Float).SetPrec(precision).Mul(root, big.NewFloat(float64(b)))
		sum.Add(sum, big.NewFloat(float64(a)))

		return sum.Quo(sum, big.NewFloat(float64(c)))
	}

	forms := [noPosition]*big.Float{
		value(0, 0, 1), value(3, -1, 6), value(-1, 1, 2), value(1, 0, 2), value(3, -1, 2), value(3, 1, 6), value(1, 0, 1),
	}

	result, _ := forms[position].Float64()

	return result
}

// stereoSource writes the generated file for the intensity ratios.
func stereoSource(t *testing.T, ratios *[noPosition]float64) []byte {
	t.Helper()

	var b bytes.Buffer

	b.WriteString("// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>\n")
	b.WriteString("// SPDX-License-Identifier: AGPL-3.0-or-later\n\n")
	b.WriteString("// Code generated by go test -run ^TestStereoTables$ -update. DO NOT EDIT.\n\n")
	b.WriteString("package layer3\n\n")
	b.WriteString("// intensityRatios holds k/(1+k) for each intensity position p from 0 to 6, where\n")
	b.WriteString("// k = tan(p π/12), rounded to the nearest float64.\n")
	b.WriteString("//\n")
	b.WriteString("// Position p scales the left channel by intensityRatios[p] and the right channel by\n")
	b.WriteString("// intensityRatios[6-p].\n")
	b.WriteString("var intensityRatios = [noPosition]float64{\n")

	for _, ratio := range ratios {
		b.WriteString(strconv.FormatFloat(ratio, 'g', -1, 64) + ",\n")
	}

	b.WriteString("}\n")

	source, err := format.Source(b.Bytes())
	require.NoError(t, err)

	return source
}
