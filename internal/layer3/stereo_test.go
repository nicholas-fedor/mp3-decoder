// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer3

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/generated"
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

// TestStereo checks joint stereo granules of every kind of block at every sampling frequency against
// the formulas, line by line, in every mode.
//
// The right channel holds values up to a random band in each window, with silent bands below it,
// and the positions include those that code none. The lower sampling frequencies take both
// intensity ratios.
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

	for at, widths := range publishedWidths {
		version := frame.MPEG1
		if at >= len(mpeg1Rates) {
			version = frame.MPEG2
		}

		for _, kind := range kinds {
			for extension := range uint8(4) {
				for trial := range 20 {
					head := frame.Header{
						SampleRate: widths.rate, Version: version, Layer: frame.Layer3,
						Mode: frame.JointStereo, Extension: extension,
					}
					places := linePlaces(&kind, widths.long, widths.short)
					left, right := randomStereo(rng, &places)
					factors := randomPositions(rng, head, kind.BlockType == ShortBlocks && kind.MixedBlock)

					kind.ScalefacCompress = trial % 2
					wantLeft, wantRight := stereoOracle(head, &left, &right, &places, &kind, factors)

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

// TestStereoLowerTopBands checks the highest long band of a lower-rate granule, which sends no
// position.
//
// It takes the position of band 20 when intensity stereo codes band 20 too, and position 0, which
// keeps the left channel's value in both channels, when band 20 lies below the bound.
func TestStereoLowerTopBands(t *testing.T) {
	t.Parallel()

	head := frame.Header{SampleRate: 22050, Version: frame.MPEG2, Layer: frame.Layer3, Mode: frame.JointStereo, Extension: 1}
	factors := &Scalefactors{}

	// Band 20 holds position 3, whose odd value scales the left channel by 2^(-2/4).
	factors.values[20], factors.widest[20] = 3, 7

	tests := []struct {
		name      string
		rightLine int
		wantLeft  float64
		wantRight float64
	}{
		{"band 20 coded too", 400, 0.5 * math.Sqrt2, 1},
		{"band 20 below the bound", 500, 1, 1},
	}

	for _, tt := range tests {
		var left, right [granuleLines]float64

		for line := range left {
			left[line] = 1
		}

		right[tt.rightLine] = 0.25

		Stereo(head, &left, &right, &GranuleInfo{}, factors)

		assert.InDelta(t, tt.wantLeft, left[575], stereoTolerance, tt.name)
		assert.InDelta(t, tt.wantRight, right[575], stereoTolerance, tt.name)
	}
}

// TestStereoLowerRatios checks the lower-rate intensity ratios of every position at both intensity
// scales against powers of 2^(-1/4) and 2^(-1/2), and that the largest value a scalefactor's width
// holds codes no position.
func TestStereoLowerRatios(t *testing.T) {
	t.Parallel()

	for scale := range 2 {
		joint := jointStereo{left: nil, right: nil, factors: nil, scale: scale, midSide: false, lowerRate: true}
		ratio := math.Pow(2, -float64(scale+1)/4)

		for position := range 15 {
			toLeft, toRight := joint.ratios(position)

			wantLeft, wantRight := 1.0, math.Pow(ratio, float64(position/2))
			if position%2 == 1 {
				wantLeft, wantRight = math.Pow(ratio, float64((position+1)/2)), 1
			}

			assert.InDelta(t, wantLeft, toLeft, 1e-15, "scale %d position %d left", scale, position)
			assert.InDelta(t, wantRight, toRight, 1e-15, "scale %d position %d right", scale, position)
		}

		assert.Equal(t, uncoded, joint.position(7, 7), "scale %d", scale)
		assert.Equal(t, uncoded, joint.position(0, 0), "scale %d", scale)
		assert.Equal(t, 6, joint.position(6, 7), "scale %d", scale)
	}
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

// randomPositions returns random scalefactors that hold intensity positions, including those that
// code none.
//
// MPEG-1 positions run from 0 to 8, so that 7 and 8 code none. At the lower sampling frequencies
// each scalefactor takes a random width from 0 to 4 bits and a value up to the largest the width
// holds, which codes none.
func randomPositions(rng *rand.Rand, head frame.Header, mixed bool) *Scalefactors {
	factors := &Scalefactors{}

	if mixed {
		factors.shift = mpeg1MixedShift
		if head.Version != frame.MPEG1 {
			factors.shift = lsfMixedShift
		}
	}

	for at := range shortSent {
		if head.Version == frame.MPEG1 {
			factors.values[at] = uint8(rng.IntN(noPosition + 2))

			continue
		}

		factors.widest[at] = uint8(1<<rng.IntN(5) - 1)
		factors.values[at] = uint8(rng.IntN(int(factors.widest[at]) + 1))
	}

	return factors
}

// stereoOracle restores joint stereo lines by the formulas.
//
// It finds the intensity bound of the long bands and of each window first, from the highest band
// in which the right channel holds a value, then restores each line on its own.
func stereoOracle(
	head frame.Header, left, right *[granuleLines]float64, places *[granuleLines]lineBand, info *GranuleInfo,
	factors *Scalefactors,
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
		position, legal := oraclePosition(head, factors, place, place.band-1 > tops[0])

		if place.short {
			coded = place.band > tops[place.window+1]
			position, legal = oraclePosition(head, factors, place, place.band-1 > tops[place.window+1])
		}

		value := left[line]

		switch {
		case head.Extension&1 != 0 && coded && legal && head.Version != frame.MPEG1:
			ratio := math.Pow(2, -float64(info.ScalefacCompress&1+1)/4)
			outLeft[line], outRight[line] = value, value

			if position%2 == 1 {
				outLeft[line] = value * math.Pow(ratio, float64((position+1)/2))
			} else {
				outRight[line] = value * math.Pow(ratio, float64(position/2))
			}
		case head.Extension&1 != 0 && coded && legal:
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

// oraclePosition returns the intensity position of a line's band, and whether it codes one.
//
// The highest long band 21 and short band 12 send no position. In MPEG-1 they take the position
// of the band below. At the lower sampling frequencies they take it only when intensity stereo
// codes the band below, and position 0 otherwise.
func oraclePosition(head frame.Header, factors *Scalefactors, place lineBand, belowCoded bool) (int, bool) {
	lower := head.Version != frame.MPEG1
	band, top := place.band, 21

	if place.short {
		top = 12
	}

	if band == top && lower && !belowCoded {
		return 0, true
	}

	band = min(band, top-1)

	value, widest := factors.Long(band), factors.widestLong(band)
	if place.short {
		value, widest = factors.Short(band, place.window), factors.widestShort(band, place.window)
	}

	if lower {
		return value, value < widest
	}

	return value, value < noPosition
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

	return generated.Source(t, "layer3", b.Bytes())
}
