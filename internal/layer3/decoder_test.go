// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/hybrid"
)

// TestHybridBlock checks the block the hybrid filterbank takes for each kind of granule.
//
// Mixed blocks hold 2 long subbands, and 4 at 8 kHz, where short band 3 starts at line 72.
func TestHybridBlock(t *testing.T) {
	t.Parallel()

	long := func(window hybrid.Window) hybrid.Block { return hybrid.Block{Long: hybrid.Subbands, Window: window} }

	tests := []struct {
		name string
		info GranuleInfo
		rate int
		want hybrid.Block
	}{
		{"normal", GranuleInfo{}, 44100, long(hybrid.Normal)},
		{"start", GranuleInfo{BlockType: StartBlock}, 44100, long(hybrid.Start)},
		{"stop", GranuleInfo{BlockType: StopBlock}, 48000, long(hybrid.Stop)},
		{"short", GranuleInfo{BlockType: ShortBlocks}, 32000, hybrid.Block{}},
		{"mixed", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 44100, hybrid.Block{Long: 2}},
		{"mixed at 22.05 kHz", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 22050, hybrid.Block{Long: 2}},
		{"mixed at 8 kHz", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 8000, hybrid.Block{Long: 4}},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, hybridBlock(&tt.info, bandsAt(tt.rate)), tt.name)
	}
}
