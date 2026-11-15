// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/hybrid"
)

// TestRoundTrip encodes random subband signals with a forward MDCT through a run of long, start,
// short and stop blocks, and checks that the filterbank gives each signal back a granule late.
//
// The encoder inverts every odd sample of every odd subband, windows each block, takes its MDCT
// scaled by 2/18 for long blocks and 2/6 for short blocks, then applies the alias butterflies of
// the encoder. Time domain aliasing must cancel between the halves of neighboring blocks.
func TestRoundTrip(t *testing.T) {
	t.Parallel()

	long := func(window hybrid.Window) hybrid.Block {
		return hybrid.Block{Long: hybrid.Subbands, Mixed: 0, Window: window}
	}
	short := hybrid.Block{Long: 0, Mixed: 0, Window: hybrid.Normal}
	run := []hybrid.Block{
		long(hybrid.Normal), long(hybrid.Normal), long(hybrid.Start), short, short, long(hybrid.Stop),
		long(hybrid.Normal), long(hybrid.Start), short, long(hybrid.Stop), long(hybrid.Normal), long(hybrid.Normal),
	}

	rng := rand.New(rand.NewPCG(31, 32))
	length := hybrid.Samples * len(run)

	var signal [hybrid.Subbands][]float64

	for subband := range signal {
		signal[subband] = make([]float64, length)

		for at := range signal[subband] {
			signal[subband][at] = 2*rng.Float64() - 1
		}
	}

	var bank hybrid.Filterbank

	for granule, block := range run {
		lines := encode(&signal, granule, block)

		var got [hybrid.Samples][hybrid.Subbands]float64

		bank.Transform(&got, &lines, block)

		for at := range hybrid.Samples {
			for subband := range hybrid.Subbands {
				want := 0.0
				if time := hybrid.Samples*(granule-1) + at; time >= 0 {
					want = signal[subband][time]
				}

				require.InDelta(t, want, got[at][subband], 1e-12, "granule %d sample %d subband %d", granule, at, subband)
			}
		}
	}
}

// encode returns the lines of one granule of subband signals in a kind of block.
//
// The granule's block covers the 36 samples from one granule before it, and samples before the
// signal starts are 0.
func encode(signal *[hybrid.Subbands][]float64, granule int, block hybrid.Block) [576]float64 {
	var lines [576]float64

	for subband := range hybrid.Subbands {
		var in [36]float64

		for i := range in {
			time := 18*(granule-1) + i
			if time < 0 {
				continue
			}

			in[i] = signal[subband][time]
			if subband%2 == 1 && time%2 == 1 {
				in[i] = -in[i]
			}
		}

		out := lines[18*subband : 18*subband+18]

		if subband < block.Long {
			for k := range 18 {
				for i := range 36 {
					out[k] += in[i] * window(block.Window, i) * math.Cos(math.Pi/72*float64((2*i+19)*(2*k+1)%144)) / 9
				}
			}

			continue
		}

		for w := range 3 {
			for k := range 6 {
				for i := range 12 {
					shape := math.Sin(math.Pi / 12 * (float64(i) + 0.5))
					out[3*k+w] += in[6+6*w+i] * shape * math.Cos(math.Pi/24*float64((2*i+7)*(2*k+1)%48)) / 3
				}
			}
		}
	}

	for boundary := 1; boundary < block.Long; boundary++ {
		for i, c := range []float64{-0.6, -0.535, -0.33, -0.185, -0.095, -0.041, -0.0142, -0.0037} {
			cs, ca := 1/math.Sqrt(1+c*c), c/math.Sqrt(1+c*c)
			lower, upper := lines[18*boundary-1-i], lines[18*boundary+i]
			lines[18*boundary-1-i] = lower*cs + upper*ca
			lines[18*boundary+i] = upper*cs - lower*ca
		}
	}

	return lines
}

// window returns sample i of a long block window by its published formula.
func window(shape hybrid.Window, i int) float64 {
	switch {
	case shape == hybrid.Start && i >= 30, shape == hybrid.Stop && i < 6:
		return 0
	case shape == hybrid.Start && i >= 24:
		return math.Sin(math.Pi / 12 * (float64(i-18) + 0.5))
	case shape == hybrid.Stop && i < 12:
		return math.Sin(math.Pi / 12 * (float64(i-6) + 0.5))
	case shape == hybrid.Start && i >= 18, shape == hybrid.Stop && i < 18:
		return 1
	default:
		return math.Sin(math.Pi / 36 * (float64(i) + 0.5))
	}
}
