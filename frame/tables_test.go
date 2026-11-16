// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBitrates checks that each bitrate row starts with free format and rises strictly, which
// catches a transposed or mistyped entry.
func TestBitrates(t *testing.T) {
	t.Parallel()

	for _, table := range [][4][15]int{mpeg1Bitrates, lsfBitrates} {
		for _, layer := range []Layer{Layer1, Layer2, Layer3} {
			row := table[layer]

			assert.Zero(t, row[0])

			for index := 2; index < len(row); index++ {
				assert.Greater(t, row[index], row[index-1], "layer %d index %d", layer, index)
			}
		}
	}
}

// TestSampleRates checks that the MPEG-2 rates are half and the MPEG-2.5 rates a quarter of the
// MPEG-1 rates.
func TestSampleRates(t *testing.T) {
	t.Parallel()

	for index, rate := range sampleRates[MPEG1] {
		assert.Equal(t, rate/2, sampleRates[MPEG2][index])
		assert.Equal(t, rate/4, sampleRates[MPEG25][index])
	}
}
