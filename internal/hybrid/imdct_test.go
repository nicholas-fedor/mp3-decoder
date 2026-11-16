// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package hybrid

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLongBlock checks the windowed IMDCT of a long block in each window against the published
// formula, computed with math.Cos and math.Sin.
func TestLongBlock(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(27, 28))
	lines := randomLines(rng)
	in := lines[:Samples]

	for _, window := range []Window{Normal, Start, Stop} {
		var got [longPoints]float64

		longBlock(&got, in, window)

		for i := range longPoints {
			var want float64

			for k := range Samples {
				want += in[k] * math.Cos(math.Pi/72*float64((2*i+19)*(2*k+1)%144))
			}

			assert.InDelta(t, want*windowOracle(window, i), got[i], tolerance, "window %d sample %d", window, i)
		}
	}
}

// TestShortBlock checks the three windowed IMDCTs of a short block against the published formula,
// each 12 samples long and 6 after the one before, with 6 silent samples at either end.
func TestShortBlock(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(29, 30))
	lines := randomLines(rng)
	in := lines[:Samples]

	got := [longPoints]float64{}
	for at := range got {
		got[at] = 99
	}

	shortBlock(&got, in)

	var want [longPoints]float64

	for window := range shortWindows {
		for i := range shortPoints {
			var sum float64

			for k := range shortPoints / 2 {
				sum += in[3*k+window] * math.Cos(math.Pi/24*float64((2*i+7)*(2*k+1)%48))
			}

			want[6+6*window+i] += sum * math.Sin(math.Pi/12*(float64(i)+0.5))
		}
	}

	for i := range longPoints {
		assert.InDelta(t, want[i], got[i], tolerance, "sample %d", i)
	}
}

// windowOracle returns sample i of a long block window by its published formula.
func windowOracle(window Window, i int) float64 {
	long := math.Sin(math.Pi / 36 * (float64(i) + 0.5))

	switch {
	case window == Start && i >= 30, window == Stop && i < 6:
		return 0
	case window == Start && i >= 24:
		return math.Sin(math.Pi / 12 * (float64(i-18) + 0.5))
	case window == Stop && i < 12:
		return math.Sin(math.Pi / 12 * (float64(i-6) + 0.5))
	case window == Start && i >= 18, window == Stop && i < 18:
		return 1
	default:
		return long
	}
}
