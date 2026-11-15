// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkTransform measures transforming one channel's granule of random lines in long blocks
// and in short blocks.
func BenchmarkTransform(b *testing.B) {
	for _, bench := range []struct {
		name  string
		block Block
	}{
		{"long", Block{Long: Subbands, Mixed: 0, Window: Normal}},
		{"short", Block{Long: 0, Mixed: 0, Window: Normal}},
	} {
		b.Run(bench.name, func(b *testing.B) {
			var (
				bank Filterbank
				out  [Samples][Subbands]float64
			)

			lines := randomLines(rand.New(rand.NewPCG(33, 34)))

			for b.Loop() {
				in := lines
				bank.Transform(&out, &in, bench.block)
			}
		})
	}
}
