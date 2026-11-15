// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

import (
	"fmt"
	"testing"
)

// FuzzTransform transforms granules of lines drawn from any bytes in any run of blocks, and checks
// every sample against the oracle that follows the published formulas.
//
// Each byte gives a line between -1 and 1, and lines past the bytes are 0. Each block byte picks
// a kind of block for one granule. The seed corpus holds a long block before a short one, mixed
// blocks after start and stop blocks, mixed start and stop blocks, and silence.
func FuzzTransform(f *testing.F) {
	f.Add([]byte{0x40, 0xc0, 0x7f, 0x81, 0x01}, []byte{0, 3})
	f.Add([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}, []byte{1, 4, 2, 5})
	f.Add([]byte{0x7f, 0x80, 0x10}, []byte{6, 4, 7, 0})
	f.Add([]byte{}, []byte{0})

	f.Fuzz(func(t *testing.T, data, kinds []byte) {
		var (
			bank   Filterbank
			oracle [Subbands][Samples]float64
		)

		for granule, kind := range kinds[:min(len(kinds), 4)] {
			var lines [granuleLines]float64

			for at := range lines {
				if at < len(data) {
					lines[at] = float64(int8(data[(at+granule)%len(data)])) / 128
				}
			}

			block := blocks[int(kind)%len(blocks)]
			want := transformOracle(&oracle, lines, block)

			var got [Samples][Subbands]float64

			bank.Transform(&got, &lines, block)
			assertSamples(t, &want, &got, fmt.Sprintf("granule %d %+v", granule, block))
		}
	})
}
