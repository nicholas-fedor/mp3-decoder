// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package hybrid

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/generated"
)

// cosineTransform holds the tables of an IMDCT of 2n lines into 4n samples, for an odd n.
type cosineTransform struct {
	// scale holds 1/(2 cos(π(2k+1)/(8n))) for each of the 2n lines.
	scale []float64
	// split holds 1/(2 cos(π(2k+1)/(4n))) for each of the n differences.
	split []float64
	// cosines holds cos(π m(2k+1)/(2n)) of the DCT of n points, for k below (n-1)/2, at
	// (n-1)/2*m+k.
	cosines []float64
}

// imdctWriter writes the statements of an unrolled IMDCT.
type imdctWriter struct {
	// source holds the statements written.
	source *bytes.Buffer
	// scale, split and cosines name the tables the statements read.
	scale, split, cosines string
	// temps counts the temporaries named.
	temps int
}

// kernelFile is the file the IMDCT kernels are generated into.
const kernelFile = "imdct_gen.go"

// Transforms of each block size.
var (
	// longTransform is the IMDCT of a long block.
	longTransform = cosineTransform{scale: longScale[:], split: longSplit[:], cosines: longCosines[:]}
	// shortTransform is the IMDCT of a short block.
	shortTransform = cosineTransform{scale: shortScale[:], split: shortSplit[:], cosines: shortCosines[:]}
)

// TestIMDCTKernels checks that the generated kernels are the IMDCT unrolled, or rewrites them with
// -update.
//
// The kernels must be the source the IMDCT unrolls to, and must give its samples bit for bit.
func TestIMDCTKernels(t *testing.T) {
	source := kernelSource(t)

	if *update {
		require.NoError(t, os.WriteFile(kernelFile, source, 0o644))

		return
	}

	t.Parallel()

	written, err := os.ReadFile(kernelFile)
	require.NoError(t, err)
	assert.Equal(t, string(source), string(written), "the kernels are stale: rerun with -update")

	rng := rand.New(rand.NewPCG(31, 32))

	for range 100 {
		lines := randomLines(rng)

		var long, longWant [longPoints]float64

		imdct36(&long, (*[Samples]float64)(lines[:Samples]))
		longTransform.imdct(longWant[:], lines[:Samples])
		assertBits(t, longWant[:], long[:], "long")

		var short, shortWant [shortPoints]float64

		imdct12(&short, (*[shortPoints / 2]float64)(lines[:shortPoints/2]))
		shortTransform.imdct(shortWant[:], lines[:shortPoints/2])
		assertBits(t, shortWant[:], short[:], "short")
	}
}

// TestIMDCT checks the IMDCT of random lines of each block size against its direct sum.
func TestIMDCT(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(33, 34))
	lines := randomLines(rng)

	for _, transform := range []cosineTransform{longTransform, shortTransform} {
		points := len(transform.scale)
		in := lines[:points]
		got := make([]float64, 2*points)

		transform.imdct(got, in)

		for i := range got {
			var want float64

			for k, line := range in {
				want += line * math.Cos(math.Pi/float64(4*points)*float64((2*i+1+points)*(2*k+1)%(8*points)))
			}

			assert.InDelta(t, want, got[i], tolerance, "%d lines, sample %d", points, i)
		}
	}
}

// imdct turns 2n lines into the 4n samples of their IMDCT, as the generated kernels unroll it.
//
// Sample i is the sum over the lines X(k) of X(k) cos(π/(8n) (2i+1+2n)(2k+1)). Those angles fold
// onto the DCT-IV of the lines, c(j), the sum over k of X(k) cos(π/(8n) (2j+1)(2k+1)). Sample i is
// c(n+i) below n, -c(3n-1-i) below 3n, and -c(i-3n) after.
//
// The DCT-IV comes from a DCT-II. Scaled by 1/(2 cos(π(2k+1)/(8n))), the lines give a DCT-II D
// with c(j) = D(j) + D(j+1), and D(2n) taken as 0. Byeong Gi Lee's split then turns the DCT-II of
// 2n points into two of n points. The DCT of the sums y(k) + y(2n-1-k) gives the even outputs. The
// DCT of the differences, scaled by 1/(2 cos(π(2k+1)/(4n))), gives the odd outputs, each of which
// adds two neighbors.
//
// Each DCT of n points folds its values in pairs, since value k and value n-1-k meet the same
// cosine, the second with the sign (-1)^m. An even output sums the pairs' sums and adds or takes
// the middle value. An odd output sums the pairs' differences, and the middle value's cosine is 0.
// No value is added to a 0 it starts from, so the unrolled kernels give the same signs of 0.
func (t *cosineTransform) imdct(dst, lines []float64) {
	points := len(lines)
	terms := points >> 1
	pairs := terms >> 1

	var sums, differences [longTerms]float64

	for k := range terms {
		low := float64(lines[k] * t.scale[k])
		high := float64(lines[points-1-k] * t.scale[points-1-k])

		sums[k] = low + high
		differences[k] = float64((low - high) * t.split[k])
	}

	var sumPairs, sumSides, differencePairs, differenceSides [longPairs]float64

	for k := range pairs {
		sumPairs[k] = sums[k] + sums[terms-1-k]
		sumSides[k] = sums[k] - sums[terms-1-k]
		differencePairs[k] = differences[k] + differences[terms-1-k]
		differenceSides[k] = differences[k] - differences[terms-1-k]
	}

	var evens, odds [longTerms]float64

	for output := range terms {
		cosines := t.cosines[pairs*output : pairs*(output+1)]
		evenValues, oddValues := &sumSides, &differenceSides

		if output&1 == 0 {
			evenValues, oddValues = &sumPairs, &differencePairs
		}

		even := float64(evenValues[0] * cosines[0])
		odd := float64(oddValues[0] * cosines[0])

		for k := 1; k < pairs; k++ {
			even += float64(evenValues[k] * cosines[k])
			odd += float64(oddValues[k] * cosines[k])
		}

		switch output & 3 {
		case 0:
			even += sums[pairs]
			odd += differences[pairs]
		case 2:
			even -= sums[pairs]
			odd -= differences[pairs]
		default:
		}

		evens[output], odds[output] = even, odd
	}

	var dct, fourth [Samples]float64

	for m := range terms {
		dct[2*m] = evens[m]
	}

	for m := range terms - 1 {
		dct[2*m+1] = odds[m] + odds[m+1]
	}

	dct[points-1] = odds[terms-1]

	for j := range points - 1 {
		fourth[j] = dct[j] + dct[j+1]
	}

	fourth[points-1] = dct[points-1]

	mirror := points + terms

	for i := range terms {
		dst[i] = fourth[terms+i]
	}

	for i := terms; i < mirror; i++ {
		dst[i] = -fourth[mirror-1-i]
	}

	for i := mirror; i < 2*points; i++ {
		dst[i] = -fourth[i-mirror]
	}
}

// kernelSource writes the generated file for the IMDCT kernels.
func kernelSource(t *testing.T) []byte {
	t.Helper()

	var b bytes.Buffer

	kernels := []struct {
		name, doc, scale, split, cosines string
		points                           int
	}{
		{"imdct36", "the 36 samples of the IMDCT of a long block's 18 lines", "longScale", "longSplit", "longCosines", Samples},
		{"imdct12", "the 12 samples of the IMDCT of a short window's 6 lines", "shortScale", "shortSplit", "shortCosines", shortPoints / 2},
	}

	for _, kernel := range kernels {
		fmt.Fprintf(&b, "// %s turns lines into %s.\n", kernel.name, kernel.doc)
		b.WriteString("//\n")
		b.WriteString("// It is the IMDCT through a DCT-IV, a DCT-II and Byeong Gi Lee's split, unrolled.\n")
		fmt.Fprintf(&b, "func %s(dst *[%d]float64, lines *[%d]float64) {\n", kernel.name, 2*kernel.points, kernel.points)

		writer := imdctWriter{source: &b, scale: kernel.scale, split: kernel.split, cosines: kernel.cosines, temps: 0}
		writer.imdct(kernel.points)

		b.WriteString("}\n\n")
	}

	return generated.Source(t, "hybrid", b.Bytes())
}

// imdct writes the statements of an IMDCT of a count of lines, in the order the IMDCT runs them.
func (w *imdctWriter) imdct(points int) {
	terms := points / 2
	pairs := terms / 2
	sums := make([]string, terms)
	differences := make([]string, terms)

	for k := range terms {
		low := w.temp(fmt.Sprintf("float64(lines[%d] * %s[%d])", k, w.scale, k))
		high := w.temp(fmt.Sprintf("float64(lines[%d] * %s[%d])", points-1-k, w.scale, points-1-k))

		sums[k] = w.temp(low + " + " + high)
		differences[k] = w.temp(fmt.Sprintf("float64((%s - %s) * %s[%d])", low, high, w.split, k))
	}

	sumPairs, sumSides := w.fold(sums, pairs)
	differencePairs, differenceSides := w.fold(differences, pairs)

	evens := make([]string, terms)
	odds := make([]string, terms)

	for output := range terms {
		evenValues, oddValues := sumSides, differenceSides
		if output%2 == 0 {
			evenValues, oddValues = sumPairs, differencePairs
		}

		even := w.temp(fmt.Sprintf("float64(%s * %s[%d])", evenValues[0], w.cosines, pairs*output))
		odd := w.temp(fmt.Sprintf("float64(%s * %s[%d])", oddValues[0], w.cosines, pairs*output))

		for k := 1; k < pairs; k++ {
			even = w.temp(fmt.Sprintf("%s + float64(%s * %s[%d])", even, evenValues[k], w.cosines, pairs*output+k))
			odd = w.temp(fmt.Sprintf("%s + float64(%s * %s[%d])", odd, oddValues[k], w.cosines, pairs*output+k))
		}

		switch output % 4 {
		case 0:
			even = w.temp(even + " + " + sums[pairs])
			odd = w.temp(odd + " + " + differences[pairs])
		case 2:
			even = w.temp(even + " - " + sums[pairs])
			odd = w.temp(odd + " - " + differences[pairs])
		default:
		}

		evens[output], odds[output] = even, odd
	}

	dct := make([]string, points)

	for m := range terms {
		dct[2*m] = evens[m]
	}

	for m := range terms - 1 {
		dct[2*m+1] = w.temp(odds[m] + " + " + odds[m+1])
	}

	dct[points-1] = odds[terms-1]

	fourth := make([]string, points)

	for j := range points - 1 {
		fourth[j] = w.temp(dct[j] + " + " + dct[j+1])
	}

	fourth[points-1] = dct[points-1]

	mirror := points + terms

	for i := range 2 * points {
		switch {
		case i < terms:
			fmt.Fprintf(w.source, "\tdst[%d] = %s\n", i, fourth[terms+i])
		case i < mirror:
			fmt.Fprintf(w.source, "\tdst[%d] = -%s\n", i, fourth[mirror-1-i])
		default:
			fmt.Fprintf(w.source, "\tdst[%d] = -%s\n", i, fourth[i-mirror])
		}
	}
}

// fold writes the sums and differences of the pairs of values k and n-1-k.
func (w *imdctWriter) fold(values []string, pairs int) ([]string, []string) {
	sums := make([]string, pairs)
	differences := make([]string, pairs)

	for k := range pairs {
		low, high := values[k], values[len(values)-1-k]
		sums[k] = w.temp(low + " + " + high)
		differences[k] = w.temp(low + " - " + high)
	}

	return sums, differences
}

// temp writes a statement that names an expression, and returns the name.
func (w *imdctWriter) temp(expression string) string {
	name := "t" + strconv.Itoa(w.temps)
	w.temps++

	fmt.Fprintf(w.source, "\t%s := %s\n", name, expression)

	return name
}

// assertBits checks that samples are those of the reference bit for bit.
func assertBits(t *testing.T, want, got []float64, name string) {
	t.Helper()

	for at := range want {
		require.Equal(t, math.Float64bits(want[at]), math.Float64bits(got[at]), "%s sample %d", name, at)
	}
}
