// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

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

	"github.com/nicholas-fedor/mp3-decoder/internal/trig"
)

// tables holds every generated table.
type tables struct {
	longWindows  [3][longPoints]float64
	longCosines  [longPoints][Samples]float64
	shortCosines [shortPoints][shortPoints / 2]float64
	shortWindow  [shortPoints]float64
	aliasCs      [aliasButterflies]float64
	aliasCa      [aliasButterflies]float64
}

// Hybrid test constants.
const (
	// generatedFile is the file the tables are generated into.
	generatedFile = "filterbank_gen.go"
	// precision is the number of bits the closed forms are computed to before rounding.
	precision = 320
	// tolerance is the largest difference the oracles may show for lines between -1 and 1.
	tolerance = 1e-12
)

// update rewrites the generated tables from their closed forms instead of checking them.
var update = flag.Bool("update", false, "rewrite the generated tables from their closed forms")

// publishedAlias holds the published alias reduction coefficients C[i].
var publishedAlias = [aliasButterflies]string{"-0.6", "-0.535", "-0.33", "-0.185", "-0.095", "-0.041", "-0.0142", "-0.0037"}

// blocks holds a block of each kind.
var blocks = []Block{
	{Long: Subbands, Mixed: 0, Window: Normal},
	{Long: Subbands, Mixed: 0, Window: Start},
	{Long: Subbands, Mixed: 0, Window: Stop},
	{Long: 0, Mixed: 0, Window: Normal},
	{Long: 2, Mixed: 2, Window: Normal},
	{Long: 4, Mixed: 4, Window: Normal},
	{Long: Subbands, Mixed: 2, Window: Start},
	{Long: Subbands, Mixed: 2, Window: Stop},
}

// TestHybridTables checks that the generated tables hold their closed forms rounded to the nearest
// float64, or rewrites them with -update.
//
// The tables must also agree with math.Sin and math.Cos. The windows must meet the Princen-Bradley
// condition, so that overlapped halves add to 1 in power, and each alias butterfly must be a
// rotation by atan(C[i]).
func TestHybridTables(t *testing.T) {
	want := closedTables()

	if *update {
		require.NoError(t, os.WriteFile(generatedFile, hybridSource(t, &want), 0o644))

		return
	}

	t.Parallel()

	assert.Equal(t, want.aliasCs, aliasCs)
	assert.Equal(t, want.aliasCa, aliasCa)
	assert.Equal(t, want.longWindows, longWindows)
	assert.Equal(t, want.shortWindow, shortWindow)
	assert.Equal(t, want.longCosines, longCosines)
	assert.Equal(t, want.shortCosines, shortCosines)

	for i, published := range publishedAlias {
		c, err := strconv.ParseFloat(published, 64)
		require.NoError(t, err)

		assert.InDelta(t, 1, aliasCs[i]*aliasCs[i]+aliasCa[i]*aliasCa[i], 1e-15, "butterfly %d", i)
		assert.InDelta(t, c, aliasCa[i]/aliasCs[i], 1e-15, "butterfly %d", i)
	}

	for i := range Samples {
		normal := longWindows[Normal]
		assert.InDelta(t, math.Sin(math.Pi/36*(float64(i)+0.5)), normal[i], 1e-15, "sample %d", i)
		assert.InDelta(t, 1, normal[i]*normal[i]+normal[i+Samples]*normal[i+Samples], 1e-15, "sample %d", i)
		assert.InDelta(t, longWindows[Start][i], longWindows[Stop][longPoints-1-i], 0, "sample %d", i)
	}

	for i := range shortStep {
		assert.InDelta(t, math.Sin(math.Pi/12*(float64(i)+0.5)), shortWindow[i], 1e-15, "sample %d", i)
		assert.InDelta(t, 1, shortWindow[i]*shortWindow[i]+shortWindow[i+shortStep]*shortWindow[i+shortStep], 1e-15, "sample %d", i)
	}

	for i, row := range longCosines {
		for k, cosine := range row {
			assert.InDelta(t, math.Cos(math.Pi/72*float64((2*i+19)*(2*k+1)%144)), cosine, 1e-15, "%d %d", i, k)
		}
	}
}

// TestTransform checks a run of granules of every kind of block against an oracle that follows
// the published formulas with math.Sin and math.Cos.
//
// Each granule's output depends on the block before it, so the run covers every pair of kinds.
func TestTransform(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(21, 22))

	var (
		bank   Filterbank
		oracle [Subbands][Samples]float64
	)

	for _, first := range blocks {
		for _, second := range blocks {
			for _, block := range []Block{first, second} {
				lines := randomLines(rng)
				want := transformOracle(&oracle, lines, block)

				var got [Samples][Subbands]float64

				bank.Transform(&got, &lines, block)
				assertSamples(t, &want, &got, fmt.Sprintf("%+v after %+v", second, first))
			}
		}
	}
}

// TestReset checks that a reset filterbank gives the samples of a fresh one.
func TestReset(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(23, 24))

	var (
		bank, fresh Filterbank
		got, want   [Samples][Subbands]float64
	)

	lines := randomLines(rng)
	bank.Transform(&got, &lines, blocks[0])
	bank.Reset()

	lines = randomLines(rng)
	again := lines

	bank.Transform(&got, &lines, blocks[0])
	fresh.Transform(&want, &again, blocks[0])

	assert.Equal(t, want, got)
}

// randomLines returns lines between -1 and 1.
func randomLines(rng *rand.Rand) [granuleLines]float64 {
	var lines [granuleLines]float64

	for at := range lines {
		lines[at] = 2*rng.Float64() - 1
	}

	return lines
}

// transformOracle transforms a granule by the published formulas, keeping its own overlap.
func transformOracle(overlap *[Subbands][Samples]float64, lines [granuleLines]float64, block Block) [Samples][Subbands]float64 {
	cs, ca := aliasOracle()

	for boundary := 1; boundary < block.Long; boundary++ {
		for i := range aliasButterflies {
			lower, upper := lines[18*boundary-1-i], lines[18*boundary+i]
			lines[18*boundary-1-i] = lower*cs[i] - upper*ca[i]
			lines[18*boundary+i] = upper*cs[i] + lower*ca[i]
		}
	}

	var out [Samples][Subbands]float64

	for subband := range Subbands {
		var z [36]float64

		in := lines[18*subband : 18*subband+18]

		if subband < block.Long {
			for i := range 36 {
				for k := range 18 {
					z[i] += in[k] * math.Cos(math.Pi/72*float64((2*i+19)*(2*k+1)%144))
				}

				window := block.Window
				if subband < block.Mixed {
					window = Normal
				}

				z[i] *= windowOracle(window, i)
			}
		} else {
			for window := range 3 {
				for i := range 12 {
					var sum float64

					for k := range 6 {
						sum += in[3*k+window] * math.Cos(math.Pi/24*float64((2*i+7)*(2*k+1)%48))
					}

					z[6+6*window+i] += sum * math.Sin(math.Pi/12*(float64(i)+0.5))
				}
			}
		}

		for i := range 18 {
			out[i][subband] = z[i] + overlap[subband][i]
			overlap[subband][i] = z[18+i]

			if subband%2 == 1 && i%2 == 1 {
				out[i][subband] = -out[i][subband]
			}
		}
	}

	return out
}

// assertSamples checks samples against those an oracle gives.
func assertSamples(t *testing.T, want, got *[Samples][Subbands]float64, name string) {
	t.Helper()

	for at := range want {
		for subband := range want[at] {
			if math.Abs(want[at][subband]-got[at][subband]) > tolerance {
				require.InDelta(t, want[at][subband], got[at][subband], tolerance, "%s sample %d subband %d", name, at, subband)
			}
		}
	}
}

// closedTables computes every generated table from its closed form.
//
// The alias butterflies come from the published coefficients as exact decimals.
func closedTables() tables {
	var out tables

	for i, published := range publishedAlias {
		c, _, err := big.ParseFloat(published, 10, precision, big.ToNearestEven)
		if err != nil {
			panic(err)
		}

		root := new(big.Float).SetPrec(precision).Mul(c, c)
		root.Add(root, big.NewFloat(1)).Sqrt(root)

		out.aliasCs[i] = rounded(new(big.Float).SetPrec(precision).Quo(big.NewFloat(1), root))
		out.aliasCa[i] = rounded(new(big.Float).SetPrec(precision).Quo(c, root))
	}

	for i := range longPoints {
		long := trig.Sin(2*i+1, 72)
		out.longWindows[Normal][i] = long

		switch {
		case i < Samples:
			out.longWindows[Start][i] = long
		case i < 24:
			out.longWindows[Start][i] = 1
		case i < 30:
			out.longWindows[Start][i] = trig.Sin(2*(i-18)+1, 24)
		}

		out.longWindows[Stop][longPoints-1-i] = out.longWindows[Start][i]

		for k := range Samples {
			out.longCosines[i][k] = trig.Cos((2*i+19)*(2*k+1), 72)
		}
	}

	for i := range shortPoints {
		out.shortWindow[i] = trig.Sin(2*i+1, 24)

		for k := range shortPoints / 2 {
			out.shortCosines[i][k] = trig.Cos((2*i+7)*(2*k+1), 24)
		}
	}

	return out
}

// rounded returns a closed form rounded to the nearest float64.
func rounded(value *big.Float) float64 {
	result, _ := value.Float64()

	return result
}

// hybridSource writes the generated file for the tables.
func hybridSource(t *testing.T, out *tables) []byte {
	t.Helper()

	var b bytes.Buffer

	row := func(values []float64) {
		for _, value := range values {
			b.WriteString(strconv.FormatFloat(value, 'g', -1, 64) + ", ")
		}

		b.WriteString("\n")
	}

	b.WriteString("// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>\n")
	b.WriteString("// SPDX-License-Identifier: AGPL-3.0-or-later\n\n")
	b.WriteString("// Code generated by go test -run ^TestHybridTables$ -update. DO NOT EDIT.\n\n")
	b.WriteString("package hybrid\n\n")
	b.WriteString("// aliasCs holds Cs[i] = 1/√(1+C[i]²) of each alias butterfly, rounded to the nearest float64.\n")
	b.WriteString("var aliasCs = [aliasButterflies]float64{\n")
	row(out.aliasCs[:])
	b.WriteString("}\n\n")
	b.WriteString("// aliasCa holds Ca[i] = C[i]/√(1+C[i]²) of each alias butterfly, rounded to the nearest float64.\n")
	b.WriteString("var aliasCa = [aliasButterflies]float64{\n")
	row(out.aliasCa[:])
	b.WriteString("}\n\n")
	b.WriteString("// longWindows holds the normal, start and stop windows of long blocks, rounded to the nearest\n")
	b.WriteString("// float64.\n")
	b.WriteString("//\n")
	b.WriteString("// The normal window is sin(π/36 (i+0.5)). The start window follows it to sample 17, holds 1 to\n")
	b.WriteString("// sample 23, falls as the second half of a short window to sample 29, and holds 0 after. The\n")
	b.WriteString("// stop window is the start window reversed.\n")
	b.WriteString("var longWindows = [3][longPoints]float64{\n")

	for _, window := range out.longWindows {
		b.WriteString("{\n")
		row(window[:])
		b.WriteString("},\n")
	}

	b.WriteString("}\n\n")
	b.WriteString("// shortWindow holds the short block window sin(π/12 (i+0.5)), rounded to the nearest float64.\n")
	b.WriteString("var shortWindow = [shortPoints]float64{\n")
	row(out.shortWindow[:])
	b.WriteString("}\n\n")
	b.WriteString("// longCosines holds cos(π/72 (2i+19)(2k+1)) of the long block IMDCT at [i][k], rounded to the\n")
	b.WriteString("// nearest float64.\n")
	b.WriteString("var longCosines = [longPoints][Samples]float64{\n")

	for _, cosines := range out.longCosines {
		b.WriteString("{\n")
		row(cosines[:])
		b.WriteString("},\n")
	}

	b.WriteString("}\n\n")
	b.WriteString("// shortCosines holds cos(π/24 (2i+7)(2k+1)) of the short block IMDCT at [i][k], rounded to the\n")
	b.WriteString("// nearest float64.\n")
	b.WriteString("var shortCosines = [shortPoints][shortPoints / 2]float64{\n")

	for _, cosines := range out.shortCosines {
		b.WriteString("{\n")
		row(cosines[:])
		b.WriteString("},\n")
	}

	b.WriteString("}\n")

	source, err := format.Source(b.Bytes())
	require.NoError(t, err)

	return source
}
