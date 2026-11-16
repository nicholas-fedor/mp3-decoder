// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSideInfoSizes checks the size of the side information of each version and mode, as the
// published layouts give it.
func TestSideInfoSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head Header
		size int
	}{
		{"MPEG-1 stereo", Header{Version: MPEG1, Layer: Layer3, Mode: JointStereo}, 32},
		{"MPEG-1 mono", Header{Version: MPEG1, Layer: Layer3, Mode: Mono}, 17},
		{"MPEG-2 stereo", Header{Version: MPEG2, Layer: Layer3, Mode: Stereo}, 17},
		{"MPEG-2.5 mono", Header{Version: MPEG25, Layer: Layer3, Mode: Mono}, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			full := SideInfo{MainDataBegin: 1, PrivateBits: 1}
			full.Granules[0][0] = Granule{WindowSwitching: true, BlockType: 2, Part23Length: 4095}

			assert.Len(t, tt.head.SideInfo(&full), tt.size)
		})
	}
}

// TestSideInfoFirstFields checks the bits of main_data_begin and the private bits at the start of
// an MPEG-1 mono frame's side information.
func TestSideInfoFirstFields(t *testing.T) {
	t.Parallel()

	side := Header{Version: MPEG1, Layer: Layer3, Mode: Mono}.SideInfo(&SideInfo{MainDataBegin: 0b1_0000_0011, PrivateBits: 0b10101})

	assert.Equal(t, []byte{0b1000_0001, 0b1101_0100}, side[:2])
}
