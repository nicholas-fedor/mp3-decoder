// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package synth

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tolerances of the tests.
const (
	// tolerance is the largest difference the oracles may show for subband samples between -1 and
	// 1.
	tolerance = 1e-12
	// impulseTolerance is the largest difference an impulse response may show, twice the largest
	// measured, 1.33e-15.
	impulseTolerance = 2.7e-15
)

// TestSynthesize checks a run of calls with random subband samples against an oracle that follows
// the published flow literally, shifting all 1024 values of its V vector on each call.
func TestSynthesize(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(41, 42))

	var (
		bank   Filterbank
		oracle [historySize]float64
	)

	for call := range 40 {
		samples := randomSamples(rng)
		want := synthesizeOracle(&oracle, &samples)

		var got [Subbands]float64

		bank.Synthesize(&got, &samples)
		assertPCM(t, &want, &got, call)
	}
}

// TestImpulse checks the impulse response of each subband.
//
// A 1 in subband k followed by silence gives D(n) cos(π/64 (16 + n mod 64)(2k+1)) for each of the
// 512 samples n the window spans, then silence.
func TestImpulse(t *testing.T) {
	t.Parallel()

	for subband := range Subbands {
		var (
			bank    Filterbank
			samples [Subbands]float64
		)

		samples[subband] = 1

		for call := range windowGroups*2 + 1 {
			var got [Subbands]float64

			bank.Synthesize(&got, &samples)
			samples = [Subbands]float64{}

			for j, value := range got {
				n := Subbands*call + j
				want := 0.0

				if n < windowTaps {
					angle := math.Pi / 64 * float64((16+n%vectorSize)*(2*subband+1)%128)
					want = window[n] * math.Cos(angle)
				}

				require.InDelta(t, want, value, impulseTolerance, "subband %d sample %d", subband, n)
			}
		}
	}
}

// TestRoundTrip passes random PCM through the published analysis filterbank and back through
// synthesis, which must give it back 481 samples late.
//
// The pair reconstructs only nearly, so the bound is twice the largest error measured, 1e-4. The
// analysis window is the synthesis window divided by 32.
func TestRoundTrip(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(43, 44))

	in := make([]float64, Subbands*400)
	for at := range in {
		in[at] = 2*rng.Float64() - 1
	}

	var (
		bank    Filterbank
		history [windowTaps]float64
	)

	out := make([]float64, 0, len(in))

	for at := 0; at < len(in); at += Subbands {
		copy(history[Subbands:], history[:windowTaps-Subbands])

		for i := range Subbands {
			history[Subbands-1-i] = in[at+i]
		}

		var (
			sums    [vectorSize]float64
			samples [Subbands]float64
			pcm     [Subbands]float64
		)

		for i := range sums {
			for group := range windowGroups {
				sums[i] += window[i+vectorSize*group] / 32 * history[i+vectorSize*group]
			}
		}

		for k := range samples {
			for i, sum := range sums {
				samples[k] += math.Cos(math.Pi/64*float64(((2*k+1)*(i-16)%128+128)%128)) * sum
			}
		}

		bank.Synthesize(&pcm, &samples)
		out = append(out, pcm[:]...)
	}

	for at := windowTaps; at < len(in); at++ {
		require.InDelta(t, in[at-481], out[at], 2e-4, "sample %d", at)
	}
}

// TestReset checks that a reset filterbank gives the samples of a fresh one.
func TestReset(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(45, 46))

	var (
		bank, fresh Filterbank
		got, want   [Subbands]float64
	)

	for range 5 {
		samples := randomSamples(rng)
		bank.Synthesize(&got, &samples)
	}

	bank.Reset()

	samples := randomSamples(rng)
	bank.Synthesize(&got, &samples)
	fresh.Synthesize(&want, &samples)

	assert.Equal(t, want, got)
}

// randomSamples returns subband samples between -1 and 1.
func randomSamples(rng *rand.Rand) [Subbands]float64 {
	var samples [Subbands]float64

	for k := range samples {
		samples[k] = 2*rng.Float64() - 1
	}

	return samples
}

// synthesizeOracle follows the published flow, keeping its own V vector.
//
// It shifts V by 64, matrixes the subband samples into V(0) to V(63), builds U from V, windows U by
// D and sums the 16 windowed values of each output sample.
func synthesizeOracle(v *[historySize]float64, samples *[Subbands]float64) [Subbands]float64 {
	copy(v[vectorSize:], v[:historySize-vectorSize])

	for i := range vectorSize {
		v[i] = 0

		for k, value := range samples {
			v[i] += value * math.Cos(math.Pi/64*float64((16+i)*(2*k+1)%128))
		}
	}

	var u [windowTaps]float64

	for i := range windowGroups {
		for j := range Subbands {
			u[64*i+j] = v[128*i+j]
			u[64*i+32+j] = v[128*i+96+j]
		}
	}

	var out [Subbands]float64

	for j := range out {
		for i := range 16 {
			out[j] += u[j+32*i] * window[j+32*i]
		}
	}

	return out
}

// assertPCM checks PCM samples against those an oracle gives.
func assertPCM(t *testing.T, want, got *[Subbands]float64, call int) {
	t.Helper()

	for j := range want {
		if math.Abs(want[j]-got[j]) > tolerance {
			require.InDelta(t, want[j], got[j], tolerance, "call %d sample %d", call, j)
		}
	}
}
