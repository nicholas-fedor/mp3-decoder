// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/frame"
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

// TestPreroll checks the frames that refill the bit reservoir, then those that refill the
// filterbanks, from the smallest frame of each stream.
func TestPreroll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head frame.Header
		want int
	}{
		// 511 bytes in frames of 104 - 38 bytes, then one frame of two granules.
		{"MPEG-1 at 44.1 kHz", frame.Header{SampleRate: 44100, Version: frame.MPEG1, Layer: frame.Layer3}, 8 + 1},
		// 511 bytes in frames of 96 - 38 bytes.
		{"MPEG-1 at 48 kHz", frame.Header{SampleRate: 48000, Version: frame.MPEG1, Layer: frame.Layer3}, 9 + 1},
		// 255 bytes in frames of 24 - 23 bytes, then two frames of one granule.
		{"MPEG-2 at 24 kHz", frame.Header{SampleRate: 24000, Version: frame.MPEG2, Layer: frame.Layer3}, 255 + 2},
		// 255 bytes in frames of 72 - 23 bytes.
		{"MPEG-2.5 at 8 kHz", frame.Header{SampleRate: 8000, Version: frame.MPEG25, Layer: frame.Layer3}, 6 + 2},
		// 511 bytes in frames of 326 - 38 bytes.
		{"free format", frame.Header{SampleRate: 44100, Bitrate: 100000, Version: frame.MPEG1, Layer: frame.Layer3, Free: true}, 2 + 1},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Preroll(tt.head), tt.name)
	}
}
