// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRegions checks the region boundaries of granules with and without window switching against
// the published band edges.
//
// At 24 kHz, long band 18 starts at line 332.
func TestRegions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		info    GranuleInfo
		rate    int
		region1 int
		region2 int
	}{
		{"long", GranuleInfo{Region0Count: 3, Region1Count: 2}, 44100, 16, 30},
		{"long to the last band", GranuleInfo{Region0Count: 15, Region1Count: 7}, 44100, 162, 576},
		{"long at 24 kHz", GranuleInfo{Region0Count: 15, Region1Count: 1}, 24000, 232, 332},
		{"short", GranuleInfo{BlockType: ShortBlocks}, 44100, 36, 576},
		{"mixed", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 48000, 36, 576},
		{"start", GranuleInfo{BlockType: StartBlock}, 32000, 36, 576},
		{"short at 22.05 kHz", GranuleInfo{BlockType: ShortBlocks}, 22050, 36, 576},
		{"stop at 22.05 kHz", GranuleInfo{BlockType: StopBlock}, 22050, 54, 576},
		{"short at 8 kHz", GranuleInfo{BlockType: ShortBlocks}, 8000, 72, 576},
		{"stop at 8 kHz", GranuleInfo{BlockType: StopBlock}, 8000, 108, 576},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			region1, region2 := regions(&tt.info, bandsAt(tt.rate))

			assert.Equal(t, tt.region1, region1)
			assert.Equal(t, tt.region2, region2)
		})
	}
}
