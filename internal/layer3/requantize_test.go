// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"bytes"
	"flag"
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
)

// update rewrites the generated tables from their closed forms instead of checking them.
var update = flag.Bool("update", false, "rewrite the generated tables from their closed forms")

// Requantization test constants.
const (
	// requantizeFile is the file the requantization tables are generated into.
	requantizeFile = "requantize_gen.go"
	// precision is the number of bits the closed forms are computed to before rounding.
	precision = 256
	// tolerance is the largest relative difference the formula oracle may show.
	tolerance = 1e-14
)

// publishedPretab holds the published pretab, transcribed apart from pretab.
var publishedPretab = [longBands]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 3, 3, 3, 2, 0}

// TestRequantizeTables checks that the generated tables hold n^(4/3) and 2^(r/4) rounded to the
// nearest float64, or rewrites them with -update.
//
// The fourth power of each cube root that is an integer must be exact.
func TestRequantizeTables(t *testing.T) {
	var (
		powers   [maxMagnitude + 1]float64
		quarters [quarterSteps]float64
	)

	for n := range powers {
		powers[n] = pow43Closed(n)
	}

	for r := range quarters {
		quarters[r] = quarterClosed(r)
	}

	if *update {
		require.NoError(t, os.WriteFile(requantizeFile, requantizeSource(t, &powers, &quarters), 0o644))

		return
	}

	t.Parallel()

	assert.Equal(t, powers, pow43)
	assert.Equal(t, quarters, quarterPowers)
	assert.Equal(t, publishedPretab, pretab)

	for root := 1; root*root*root <= maxMagnitude; root++ {
		assert.InDelta(t, float64(root*root*root*root), pow43[root*root*root], 0, "%d^3", root)
	}
}

// TestRequantize checks requantized granules of every kind of block at every sampling frequency
// against the formula, line by line.
//
// The formula maps each line to its band through the published band widths, and computes each
// factor with math.Pow. Mixed blocks hold long bands up to where short band 3 starts.
func TestRequantize(t *testing.T) {
	t.Parallel()

	kinds := []GranuleInfo{
		{},
		{BlockType: StartBlock},
		{BlockType: StopBlock},
		{BlockType: ShortBlocks},
		{BlockType: ShortBlocks, MixedBlock: true},
	}

	rng := rand.New(rand.NewPCG(11, 12))

	for _, widths := range publishedWidths {
		for _, kind := range kinds {
			for _, scale := range []bool{false, true} {
				for _, preflag := range []bool{false, true} {
					info := kind
					info.GlobalGain = rng.IntN(256)
					info.ScalefacScale = scale
					info.SubblockGain = [3]int{rng.IntN(8), rng.IntN(8), rng.IntN(8)}

					spectrum, factors := randomGranule(rng, widths.rate, info.BlockType == ShortBlocks && info.MixedBlock, preflag)

					var got [granuleLines]float64

					spectrum.Requantize(&got, &info, factors)

					want := formula(spectrum, &info, factors, widths.long, widths.short)
					name := fmt.Sprintf("%d Hz %+v", widths.rate, info)

					for line := range got {
						if want[line] == 0 {
							require.Zero(t, got[line], "%s line %d", name, line)

							continue
						}

						require.InEpsilon(t, want[line], got[line], tolerance, "%s line %d", name, line)
					}
				}
			}
		}
	}
}

// TestRequantizeExtremes checks the largest magnitude at the highest and lowest gains, and a value
// of 1 at gain 1.
func TestRequantizeExtremes(t *testing.T) {
	t.Parallel()

	var (
		spectrum Spectrum
		factors  Scalefactors
		got      [granuleLines]float64
	)

	spectrum.bands = bandsAt(44100)
	spectrum.Values[0], spectrum.Values[1], spectrum.Decoded = 1, -maxMagnitude, 2

	spectrum.Requantize(&got, &GranuleInfo{GlobalGain: gainOffset}, &factors)
	assert.Equal(t, [2]float64{1, -pow43Closed(maxMagnitude)}, [2]float64{got[0], got[1]})

	spectrum.Requantize(&got, &GranuleInfo{GlobalGain: 255}, &factors)
	assert.InEpsilon(t, -math.Pow(maxMagnitude, 4.0/3)*math.Pow(2, 45.0/4), got[1], tolerance)

	factors.values[0] = 15
	spectrum.Requantize(&got, &GranuleInfo{BlockType: ShortBlocks, SubblockGain: [3]int{7, 7, 7}, ScalefacScale: true}, &factors)
	assert.InEpsilon(t, math.Pow(2, (-210.0-56-60)/4), got[0], tolerance)
}

// TestGain checks the gain at the ends of its range and around 0 against math.Pow.
func TestGain(t *testing.T) {
	t.Parallel()

	for _, quarters := range []int{-1298, -5, -4, -1, 0, 1, 3, 4, 45} {
		assert.InEpsilon(t, math.Pow(2, float64(quarters)/4), gain(quarters), tolerance, "%d", quarters)
	}
}

// randomGranule returns random quantized values up to a random count, with random scalefactors.
func randomGranule(rng *rand.Rand, rate int, mixed, preflag bool) (*Spectrum, *Scalefactors) {
	spectrum := &Spectrum{bands: bandsAt(rate), Decoded: 2 * rng.IntN(granuleLines/2+1)}

	for line := range spectrum.Decoded {
		magnitude := rng.IntN(16)
		if rng.IntN(50) == 0 {
			magnitude = rng.IntN(maxMagnitude + 1)
		}

		if rng.IntN(2) == 0 {
			magnitude = -magnitude
		}

		spectrum.Values[line] = magnitude
	}

	factors := &Scalefactors{mixed: mixed, preflag: preflag}

	for at := range shortSent {
		factors.values[at] = uint8(rng.IntN(16))
	}

	return spectrum, factors
}

// formula requantizes each line by the published formula, finding its band and window from band
// widths.
func formula(spectrum *Spectrum, info *GranuleInfo, factors *Scalefactors, long [longBands]int, short [shortBands]int) [granuleLines]float64 {
	var out [granuleLines]float64

	multiplier := 0.5
	if info.ScalefacScale {
		multiplier = 1
	}

	global := math.Pow(2, 0.25*float64(info.GlobalGain-210))
	line := 0

	mixedEnd := 3 * (short[0] + short[1] + short[2])

	if info.BlockType != ShortBlocks || info.MixedBlock {
		for band, width := range long {
			if info.BlockType == ShortBlocks && line >= mixedEnd {
				break
			}

			factor := float64(factors.Long(band))
			if factors.preflag {
				factor += float64(publishedPretab[band])
			}

			for range width {
				out[line] = global * math.Pow(2, -multiplier*factor) * magnitude(spectrum.Values[line])
				line++
			}
		}
	}

	if info.BlockType == ShortBlocks {
		start := 0

		for band, width := range short {
			start += 3 * width
			if start <= line {
				continue
			}

			for window := range 3 {
				gain := math.Pow(2, -2*float64(info.SubblockGain[window])) * math.Pow(2, -multiplier*float64(factors.Short(band, window)))

				for range width {
					out[line] = global * gain * magnitude(spectrum.Values[line])
					line++
				}
			}
		}
	}

	return out
}

// magnitude returns sign(v) |v|^(4/3).
func magnitude(value int) float64 {
	return math.Copysign(math.Pow(math.Abs(float64(value)), 4.0/3), float64(value))
}

// pow43Closed returns n^(4/3) rounded to the nearest float64, from a cube root found by Newton's
// method to 256 bits.
func pow43Closed(n int) float64 {
	if n == 0 {
		return 0
	}

	value := new(big.Float).SetPrec(precision).SetInt64(int64(n))
	root := new(big.Float).SetPrec(precision).SetFloat64(math.Cbrt(float64(n)))
	three := new(big.Float).SetPrec(precision).SetInt64(3)

	for range 8 {
		square := new(big.Float).SetPrec(precision).Mul(root, root)
		next := new(big.Float).SetPrec(precision).Quo(value, square)
		next.Add(next, root)
		next.Add(next, root)
		root.Quo(next, three)
	}

	result, _ := new(big.Float).SetPrec(precision).Mul(value, root).Float64()

	return result
}

// quarterClosed returns 2^(r/4) rounded to the nearest float64, as the square root of the square
// root of 2^r.
func quarterClosed(r int) float64 {
	value := new(big.Float).SetPrec(precision).SetMantExp(big.NewFloat(1), r)
	value.Sqrt(value)
	value.Sqrt(value)

	result, _ := value.Float64()

	return result
}

// requantizeSource writes the generated file for the requantization tables.
func requantizeSource(t *testing.T, powers *[maxMagnitude + 1]float64, quarters *[quarterSteps]float64) []byte {
	t.Helper()

	var b bytes.Buffer

	b.WriteString("// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>\n")
	b.WriteString("// SPDX-License-Identifier: AGPL-3.0-or-later\n\n")
	b.WriteString("// Code generated by go test -run ^TestRequantizeTables$ -update. DO NOT EDIT.\n\n")
	b.WriteString("package layer3\n\n")
	b.WriteString("// quarterPowers holds 2^(r/4) for r from 0 to 3, rounded to the nearest float64.\n")
	b.WriteString("var quarterPowers = [quarterSteps]float64{\n")

	for _, power := range quarters {
		b.WriteString(strconv.FormatFloat(power, 'g', -1, 64) + ",\n")
	}

	b.WriteString("}\n\n")
	b.WriteString("// pow43 holds n^(4/3) for each quantized magnitude n, rounded to the nearest float64.\n")
	b.WriteString("var pow43 = [maxMagnitude + 1]float64{\n")

	for at := 0; at < len(powers); at += 6 {
		for _, power := range powers[at:min(at+6, len(powers))] {
			b.WriteString(strconv.FormatFloat(power, 'g', -1, 64) + ", ")
		}

		b.WriteString("\n")
	}

	b.WriteString("}\n")

	source, err := format.Source(b.Bytes())
	require.NoError(t, err)

	return source
}
