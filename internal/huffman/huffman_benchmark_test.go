// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// granuleLines is the number of spectral values in one granule.
const granuleLines = 576

// randomBits returns 64 KiB of random bits.
//
// A Huffman decoder reading random bits gives each value with the probability 2^-length its code
// word's length implies, which is the distribution the code was designed for.
func randomBits() []byte {
	rng := rand.New(rand.NewPCG(5, 6))
	data := make([]byte, 64<<10)

	for i := range data {
		data[i] = byte(rng.Uint32())
	}

	return data
}

// BenchmarkBigValues measures decoding a granule's worth of pairs at a time in tables of each size,
// with and without linbits.
func BenchmarkBigValues(b *testing.B) {
	data := randomBits()
	values := make([]int, granuleLines)

	for _, table := range []int{1, 7, 13, 15, 16, 24} {
		b.Run(fmt.Sprintf("table %d", table), func(b *testing.B) {
			var r bits.Reader

			pairs := 0

			for b.Loop() {
				r.Reset(data, 0)

				for r.Pos() < len(data)*8-granuleLines*maxPairBits {
					BigValues(&r, table, values)

					pairs += granuleLines / 2
				}
			}

			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(pairs), "ns/pair")
		})
	}
}

// BenchmarkCount1 measures decoding quadruples in tables A and B.
func BenchmarkCount1(b *testing.B) {
	data := randomBits()
	values := make([]int, granuleLines)

	for table, name := range []string{"A", "B"} {
		b.Run("table "+name, func(b *testing.B) {
			var r bits.Reader

			for b.Loop() {
				r.Reset(data, 0)

				for r.Pos() < len(data)*8-granuleLines*maxQuadBits {
					Count1(&r, table, values, len(data)*8)
				}
			}
		})
	}
}

// BenchmarkLookups compares decoding code words of table 13, whose code words run longest,
// through the generated lookup tables and through the same tables built at run time.
func BenchmarkLookups(b *testing.B) {
	data := randomBits()
	ref := bigValueLookups[13]

	built := new([lookupSize]uint16)
	copy(built[:], buildLookups().entries)

	for _, variant := range []struct {
		name    string
		entries *[lookupSize]uint16
	}{
		{"generated", &lookupEntries},
		{"built", built},
	} {
		b.Run(variant.name, func(b *testing.B) {
			var r bits.Reader

			for b.Loop() {
				r.Reset(data, 0)

				for r.Pos() < len(data)*8-maxLen {
					readCode(variant.entries, ref, &r)
				}
			}
		})
	}
}

// BenchmarkBuildLookups measures building every lookup table at run time, the startup cost the
// generated tables avoid.
func BenchmarkBuildLookups(b *testing.B) {
	for b.Loop() {
		buildLookups()
	}
}
