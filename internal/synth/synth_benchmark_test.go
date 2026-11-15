// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package synth

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkSynthesize measures synthesizing one channel's granule, 18 calls of random subband
// samples, so each operation is one granule.
func BenchmarkSynthesize(b *testing.B) {
	rng := rand.New(rand.NewPCG(47, 48))

	var (
		bank    Filterbank
		samples [18][Subbands]float64
		pcm     [Subbands]float64
	)

	for at := range samples {
		samples[at] = randomSamples(rng)
	}

	b.ReportAllocs()

	for b.Loop() {
		for at := range samples {
			bank.Synthesize(&pcm, &samples[at])
		}
	}
}
