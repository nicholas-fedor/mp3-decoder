// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// BenchmarkFind measures a search through 64 KiB of random bytes to the two frames after them.
func BenchmarkFind(b *testing.B) {
	noise := make([]byte, 64<<10)
	rng := rand.New(rand.NewPCG(1, 2))

	for i := range noise {
		noise[i] = byte(rng.Uint32())
	}

	window := slices.Concat(noise, mpeg1Head.Frame(417), mpeg1Head.Frame(417))

	b.SetBytes(int64(len(window)))

	for b.Loop() {
		if _, _, confirmed := Find(window, nil); !confirmed {
			b.Fatal("no frame found")
		}
	}
}
