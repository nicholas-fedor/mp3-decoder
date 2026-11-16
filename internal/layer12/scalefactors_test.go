// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"bytes"
	"flag"
	"math"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/generated"
)

// update rewrites the generated tables instead of checking them.
var update = flag.Bool("update", false, "rewrite the generated tables from their closed forms")

// Table generation.
const (
	// scalefactorFile is the generated file for the scalefactor table.
	scalefactorFile = "scalefactors_gen.go"
	// precision is the number of bits the closed forms are computed to before rounding.
	precision = 256
	// newtonSteps is the number of Newton steps that take a float64 cube root to precision.
	newtonSteps = 4
)

// TestScalefactorTable checks the scalefactor table against its closed form, and rewrites the
// generated file with -update.
//
// Every third scalefactor is a power of two and must be exact. The last index is not used, and
// its scalefactor is 0.
func TestScalefactorTable(t *testing.T) {
	var factors [scalefactorIndices]float64

	for index := range scalefactorIndices - 1 {
		factors[index] = scalefactorClosed(index)
	}

	if *update {
		require.NoError(t, os.WriteFile(scalefactorFile, scalefactorSource(t, &factors), 0o644))

		return
	}

	t.Parallel()

	assert.Equal(t, factors, scalefactors)

	for index := 0; index < scalefactorIndices-1; index += 3 {
		assert.InDelta(t, math.Ldexp(1, 1-index/3), scalefactors[index], 0, "index %d", index)
	}

	for index := range scalefactorIndices - 1 {
		assert.InEpsilon(t, math.Pow(2, 1-float64(index)/3), scalefactors[index], 1e-15, "index %d", index)
	}

	assert.Zero(t, scalefactors[scalefactorIndices-1])
}

// scalefactorClosed returns 2^(1-i/3) rounded to the nearest float64, as the cube root of
// 2^(3-i) by Newton's method.
func scalefactorClosed(index int) float64 {
	value := new(big.Float).SetPrec(precision).SetMantExp(big.NewFloat(1), 3-index)
	root := new(big.Float).SetPrec(precision).SetFloat64(math.Cbrt(math.Ldexp(1, 3-index)))
	two := new(big.Float).SetPrec(precision).SetInt64(2)
	three := new(big.Float).SetPrec(precision).SetInt64(3)

	for range newtonSteps {
		square := new(big.Float).SetPrec(precision).Mul(root, root)
		next := new(big.Float).SetPrec(precision).Quo(value, square)
		next.Add(next, new(big.Float).SetPrec(precision).Mul(two, root))
		root = next.Quo(next, three)
	}

	result, _ := root.Float64()

	return result
}

// scalefactorSource writes the generated file for the scalefactor table.
func scalefactorSource(t *testing.T, factors *[scalefactorIndices]float64) []byte {
	t.Helper()

	var b bytes.Buffer

	b.WriteString("// scalefactors holds 2^(1-i/3) for each scalefactor index i, rounded to the nearest float64, and\n")
	b.WriteString("// 0 for the last index, which is not used.\n")
	b.WriteString("var scalefactors = [scalefactorIndices]float64{\n")

	for at := 0; at < len(factors); at += 4 {
		for _, factor := range factors[at:min(at+4, len(factors))] {
			b.WriteString(strconv.FormatFloat(factor, 'g', -1, 64) + ", ")
		}

		b.WriteString("\n")
	}

	b.WriteString("}\n")

	return generated.Source(t, "layer12", b.Bytes())
}
