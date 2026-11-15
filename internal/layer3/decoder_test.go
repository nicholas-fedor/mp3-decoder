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
// Mixed blocks hold 2 long subbands, and 4 at 8 kHz, where short band 3 starts at line 72. A start
// or stop block with the mixed flag gives as many subbands the normal window.
func TestHybridBlock(t *testing.T) {
	t.Parallel()

	block := func(long, mixed int, window hybrid.Window) hybrid.Block {
		return hybrid.Block{Long: long, Mixed: mixed, Window: window}
	}

	tests := []struct {
		name string
		info GranuleInfo
		rate int
		want hybrid.Block
	}{
		{"normal", GranuleInfo{}, 44100, block(32, 0, hybrid.Normal)},
		{"start", GranuleInfo{BlockType: StartBlock}, 44100, block(32, 0, hybrid.Start)},
		{"stop", GranuleInfo{BlockType: StopBlock}, 48000, block(32, 0, hybrid.Stop)},
		{"short", GranuleInfo{BlockType: ShortBlocks}, 32000, block(0, 0, hybrid.Normal)},
		{"mixed", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 44100, block(2, 2, hybrid.Normal)},
		{"mixed at 22.05 kHz", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 22050, block(2, 2, hybrid.Normal)},
		{"mixed at 8 kHz", GranuleInfo{BlockType: ShortBlocks, MixedBlock: true}, 8000, block(4, 4, hybrid.Normal)},
		{"mixed start", GranuleInfo{BlockType: StartBlock, MixedBlock: true}, 44100, block(32, 2, hybrid.Start)},
		{"mixed stop", GranuleInfo{BlockType: StopBlock, MixedBlock: true}, 8000, block(32, 4, hybrid.Stop)},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, hybridBlock(&tt.info, bandsAt(tt.rate)), tt.name)
	}
}
