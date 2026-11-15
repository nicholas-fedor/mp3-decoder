// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package trig

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExactValues checks angles whose sines and cosines have closed forms, which must round
// exactly.
func TestExactValues(t *testing.T) {
	t.Parallel()

	half := new(big.Float).SetPrec(precision).SetInt64(2)
	half.Sqrt(half).Quo(half, big.NewFloat(2))
	root, _ := half.Float64()

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"cos 0", Cos(0, 1), 1},
		{"cos π", Cos(1, 1), -1},
		{"cos π/2", Cos(1, 2), 0},
		{"cos 3π/2", Cos(3, 2), 0},
		{"cos -π/2", Cos(-1, 2), 0},
		{"cos π/3", Cos(1, 3), 0.5},
		{"cos 2π/3", Cos(2, 3), -0.5},
		{"cos π/4", Cos(1, 4), root},
		{"cos -7π/4", Cos(-7, 4), root},
		{"cos 37π/4", Cos(37, 4), -root},
		{"sin π/6", Sin(1, 6), 0.5},
		{"sin π", Sin(1, 1), 0},
		{"sin -π/2", Sin(-1, 2), -1},
		{"sin 3π/4", Sin(3, 4), root},
		{"sec 0", Sec(0, 1), 1},
		{"sec π/3", Sec(1, 3), 2},
		{"sec 2π/3", Sec(2, 3), -2},
		{"sec π/4", Sec(1, 4), math.Sqrt2},
	}

	for _, tt := range tests {
		assert.InDelta(t, tt.want, tt.got, 0, tt.name)
	}
}

// TestMath checks many angles against package math, at angles reduced to one period so that its
// own rounding of the angle stays small, and each secant against its cosine.
func TestMath(t *testing.T) {
	t.Parallel()

	for _, d := range []int{12, 24, 64, 72} {
		for n := -3 * d; n <= 3*d; n++ {
			angle := math.Pi * float64((n%(2*d)+2*d)%(2*d)) / float64(d)
			assert.InDelta(t, math.Cos(angle), Cos(n, d), 1e-15, "cos %dπ/%d", n, d)
			assert.InDelta(t, math.Sin(angle), Sin(n, d), 1e-15, "sin %dπ/%d", n, d)

			// Each rounds to the nearest float64, so their product is 1 within two rounding errors.
			if Cos(n, d) != 0 {
				assert.InDelta(t, 1, Sec(n, d)*Cos(n, d), 4.5e-16, "sec %dπ/%d", n, d)
			}
		}
	}
}

// TestSecZero checks that the secant is infinite where the cosine is 0.
func TestSecZero(t *testing.T) {
	t.Parallel()

	assert.True(t, math.IsInf(Sec(1, 2), 1))
}

// TestPi checks π against the float64 constant, which is π rounded to the nearest float64.
func TestPi(t *testing.T) {
	t.Parallel()

	value, _ := pi().Float64()
	assert.InDelta(t, math.Pi, value, 0)
}
