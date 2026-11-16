// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package hybrid

import (
	"math"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReduceAliases checks alias reduction between each count of the lowest subbands.
//
// Only the 8 lines on either side of a boundary between those subbands may change, and the
// encoder's butterflies, the same rotations turned back, must restore the lines.
func TestReduceAliases(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(25, 26))
	cs, ca := aliasOracle()

	for _, long := range []int{0, 1, 2, 4, Subbands} {
		original := randomLines(rng)
		lines := original

		reduceAliases(&lines, long)

		for line := range lines {
			boundary := (line + Samples/2) / Samples
			near := line%Samples < aliasButterflies || line%Samples >= Samples-aliasButterflies

			if boundary == 0 || boundary >= long || !near {
				assert.InDelta(t, original[line], lines[line], 0, "long %d line %d", long, line)
			}
		}

		for boundary := 1; boundary < long; boundary++ {
			for i := range aliasButterflies {
				lower, upper := lines[Samples*boundary-1-i], lines[Samples*boundary+i]
				lines[Samples*boundary-1-i] = lower*cs[i] + upper*ca[i]
				lines[Samples*boundary+i] = upper*cs[i] - lower*ca[i]
			}
		}

		for line := range lines {
			assert.InDelta(t, original[line], lines[line], 1e-15, "long %d line %d", long, line)
		}
	}
}

// aliasOracle returns the alias butterflies Cs[i] = 1/√(1+C[i]²) and Ca[i] = C[i]/√(1+C[i]²).
func aliasOracle() ([aliasButterflies]float64, [aliasButterflies]float64) {
	var cs, ca [aliasButterflies]float64

	for i, published := range publishedAlias {
		c, _ := strconv.ParseFloat(published, 64)
		cs[i], ca[i] = 1/math.Sqrt(1+c*c), c/math.Sqrt(1+c*c)
	}

	return cs, ca
}
