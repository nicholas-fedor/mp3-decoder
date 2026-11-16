// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package synth

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

// kernelFile is the file the 32-point DCT kernel is generated into.
const kernelFile = "dct_gen.go"

// dctScratch is the number of values the recursive DCT keeps while it runs, 32 + 16 + 8 + 4 + 2.
const dctScratch = 2*Subbands - 2

// kernel writes the statements of an unrolled DCT.
type kernel struct {
	// source holds the statements written.
	source *bytes.Buffer
	// temps counts the temporaries named.
	temps int
}

// TestDCTKernel checks that the generated 32-point kernel is the unrolled recursion, or rewrites it
// with -update.
//
// The kernel must be the source the recursion unrolls to, and must give its outputs bit for bit.
func TestDCTKernel(t *testing.T) {
	source := kernelSource(t)

	if *update {
		require.NoError(t, os.WriteFile(kernelFile, source, 0o644))

		return
	}

	t.Parallel()

	written, err := os.ReadFile(kernelFile)
	require.NoError(t, err)
	assert.Equal(t, string(source), string(written), "the kernel is stale: rerun with -update")

	rng := rand.New(rand.NewPCG(51, 52))

	for range 100 {
		values := randomSamples(rng)
		want := values

		var scratch [dctScratch]float64

		dct(want[:], scratch[:])
		dct32(&values)

		for m := range values {
			require.Equal(t, math.Float64bits(want[m]), math.Float64bits(values[m]), "output %d", m)
		}
	}
}

// TestDCT checks the DCT of random values of every length it takes against the direct sum of
// x(k) cos(π m(2k+1)/(2n)).
func TestDCT(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(49, 50))

	for points := 2; points <= Subbands; points *= 2 {
		values := make([]float64, points)
		for k := range values {
			values[k] = 2*rng.Float64() - 1
		}

		want := make([]float64, points)

		for m := range want {
			for k, value := range values {
				want[m] += value * math.Cos(math.Pi*float64(m*(2*k+1))/float64(2*points))
			}
		}

		var scratch [dctScratch]float64

		dct(values, scratch[:])

		for m := range want {
			assert.InDelta(t, want[m], values[m], 1e-13, "%d points, output %d", points, m)
		}
	}
}

// kernelSource writes the generated file for the 32-point DCT kernel.
func kernelSource(t *testing.T) []byte {
	t.Helper()

	var b bytes.Buffer

	b.WriteString("// dct32 turns 32 values into their DCT in place, X(m), the sum over k of x(k) cos(π m(2k+1)/64).\n")
	b.WriteString("//\n")
	b.WriteString("// It is Byeong Gi Lee's algorithm unrolled. Each split of n values takes their sums\n")
	b.WriteString("// x(k) + x(n-1-k) for the even outputs and their differences scaled by 1/(2 cos(π(2k+1)/(2n)))\n")
	b.WriteString("// for the odd outputs, each of which adds two neighboring outputs of the differences' DCT.\n")
	b.WriteString("func dct32(values *[Subbands]float64) {\n")

	unrolled := kernel{source: &b, temps: 0}

	inputs := make([]string, Subbands)
	for k := range inputs {
		inputs[k] = "values[" + strconv.Itoa(k) + "]"
	}

	for m, output := range unrolled.dct(inputs) {
		fmt.Fprintf(&b, "\tvalues[%d] = %s\n", m, output)
	}

	b.WriteString("}\n")

	return generated.Source(t, "synth", b.Bytes())
}

// dct writes the statements of a DCT of the named values, in the order the recursion runs them,
// and returns the names of its outputs.
func (k *kernel) dct(values []string) []string {
	points := len(values)
	half := points / 2
	sums := make([]string, half)
	differences := make([]string, half)

	for at := range half {
		low, high := values[at], values[points-1-at]
		sums[at] = k.temp(low + " + " + high)
		differences[at] = k.temp(fmt.Sprintf("float64((%s - %s) * lee[%d])", low, high, Subbands-points+at))
	}

	if half > 1 {
		sums = k.dct(sums)
		differences = k.dct(differences)
	}

	out := make([]string, points)

	for m := range half - 1 {
		out[2*m] = sums[m]
		out[2*m+1] = k.temp(differences[m] + " + " + differences[m+1])
	}

	out[points-2] = sums[half-1]
	out[points-1] = differences[half-1]

	return out
}

// temp writes a statement that names an expression, and returns the name.
func (k *kernel) temp(expression string) string {
	name := "t" + strconv.Itoa(k.temps)
	k.temps++

	fmt.Fprintf(k.source, "\t%s := %s\n", name, expression)

	return name
}

// dct turns values into their DCT in place, by Byeong Gi Lee's algorithm, as the recursion that
// the generated 32-point kernel unrolls.
//
// Of n values x(k), it gives X(m), the sum over k of x(k) cos(π m(2k+1)/(2n)). It splits them into
// the sums g(k) = x(k) + x(n-1-k) and the scaled differences h(k) = (x(k) - x(n-1-k)) /
// (2 cos(π(2k+1)/(2n))), whose DCTs of n/2 points G and H give X(2m) = G(m) and
// X(2m+1) = H(m) + H(m+1), with H(n/2) taken as 0.
//
// Parameters:
//   - values: the values, a power of two of them up to 32.
//   - scratch: room for the values of every split, 2n - 2 of them.
func dct(values, scratch []float64) {
	points := len(values)
	half := points >> 1
	split := scratch[:points]
	factors := lee[Subbands-points : Subbands-half]

	for k := range half {
		a, b := values[k], values[points-1-k]

		split[k] = a + b
		split[half+k] = float64((a - b) * factors[k])
	}

	if half > 1 {
		dct(split[:half], scratch[points:])
		dct(split[half:], scratch[points:])
	}

	for m := range half - 1 {
		values[2*m] = split[m]
		values[2*m+1] = split[half+m] + split[half+m+1]
	}

	values[points-2] = split[half-1]
	values[points-1] = split[points-1]
}
