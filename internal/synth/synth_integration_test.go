// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package synth_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// TestLinear checks that synthesis is linear and time invariant over a run of calls.
//
// The output of a sum of two runs, one scaled, must be the same sum of their outputs. A run that
// starts after calls of silence must give the same output, those calls late.
func TestLinear(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(49, 50))

	first, second := randomRun(rng, 30), randomRun(rng, 30)
	mixed := make([][synth.Subbands]float64, len(first))

	for call := range mixed {
		for k := range mixed[call] {
			mixed[call][k] = first[call][k] - 0.5*second[call][k]
		}
	}

	a, b, sum := synthesize(first), synthesize(second), synthesize(mixed)

	for at := range sum {
		require.InDelta(t, a[at]-0.5*b[at], sum[at], 1e-12, "sample %d", at)
	}

	late := synthesize(append(make([][synth.Subbands]float64, 7), first...))

	for at := range a {
		require.InDelta(t, a[at], late[7*synth.Subbands+at], 1e-15, "sample %d", at)
	}
}

// TestSilence checks that silence gives silence, and that a reset filterbank forgets what came
// before.
func TestSilence(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(51, 52))

	var (
		bank    synth.Filterbank
		silence [synth.Subbands]float64
		pcm     [synth.Subbands]float64
	)

	bank.Synthesize(&pcm, &silence)
	assert.Equal(t, silence, pcm)

	for _, samples := range randomRun(rng, 5) {
		bank.Synthesize(&pcm, &samples)
	}

	assert.NotEqual(t, silence, pcm)

	bank.Reset()
	bank.Synthesize(&pcm, &silence)
	assert.Equal(t, silence, pcm)
}

// randomRun returns calls of subband samples between -1 and 1.
func randomRun(rng *rand.Rand, calls int) [][synth.Subbands]float64 {
	run := make([][synth.Subbands]float64, calls)

	for call := range run {
		for k := range run[call] {
			run[call][k] = 2*rng.Float64() - 1
		}
	}

	return run
}

// synthesize returns the PCM samples a fresh filterbank gives for a run of calls.
func synthesize(run [][synth.Subbands]float64) []float64 {
	var bank synth.Filterbank

	out := make([]float64, 0, len(run)*synth.Subbands)

	for _, samples := range run {
		var pcm [synth.Subbands]float64

		bank.Synthesize(&pcm, &samples)
		out = append(out, pcm[:]...)
	}

	return out
}
