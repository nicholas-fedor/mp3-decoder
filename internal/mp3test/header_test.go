// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHeaderBytes checks encoded headers against words worked out bit by bit.
func TestHeaderBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head Header
		want []byte
	}{
		{"MPEG-1 Layer III", Header{Version: MPEG1, Layer: Layer3, Bitrate: 9}, []byte{0xff, 0xfb, 0x90, 0x00}},
		{"protected", Header{Version: MPEG1, Layer: Layer3, Bitrate: 9, Protected: true}, []byte{0xff, 0xfa, 0x90, 0x00}},
		{"padded", Header{Version: MPEG1, Layer: Layer3, Bitrate: 9, Padding: true}, []byte{0xff, 0xfb, 0x92, 0x00}},
		{
			"MPEG-2 mono",
			Header{Version: MPEG2, Layer: Layer3, Bitrate: 8, Mode: Mono},
			[]byte{0xff, 0xf3, 0x80, 0xc0},
		},
		{
			"joint stereo with emphasis",
			Header{Version: MPEG1, Layer: Layer3, Bitrate: 9, Mode: JointStereo, Extension: 2, Emphasis: 1},
			[]byte{0xff, 0xfb, 0x90, 0x61},
		},
		{"MPEG-2.5 Layer I", Header{Version: MPEG25, Layer: Layer1, Rate: 2}, []byte{0xff, 0xe7, 0x08, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.head.Bytes())
		})
	}
}

// TestHeaderFrame checks that a frame starts with its header and holds zeros after it.
func TestHeaderFrame(t *testing.T) {
	t.Parallel()

	head := Header{Version: MPEG1, Layer: Layer3, Bitrate: 9}
	frame := head.Frame(417)

	assert.Len(t, frame, 417)
	assert.Equal(t, head.Bytes(), frame[:4])
	assert.Equal(t, make([]byte, 413), frame[4:])
	assert.Len(t, head.Frame(2), 4)
}

// TestPlace checks that a payload lands at its offset.
func TestPlace(t *testing.T) {
	t.Parallel()

	frame := Place(make([]byte, 6), 2, []byte{1, 2})

	assert.Equal(t, []byte{0, 0, 1, 2, 0, 0}, frame)
}
