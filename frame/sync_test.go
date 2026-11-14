// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Headers the sync tests build frames from.
var (
	// mpeg1Frame is an unprotected MPEG-1 Layer III header at 128 kbit/s and 44.1 kHz, whose frames
	// hold 417 bytes.
	mpeg1Frame = fields{version: rawMPEG1, layer: rawLayer3, bitrate: 9, unprotected: true}
	// mpeg2Frame is an unprotected MPEG-2 Layer III header at 64 kbit/s and 22.05 kHz.
	mpeg2Frame = fields{version: rawMPEG2, layer: rawLayer3, bitrate: 8, unprotected: true}
	// layer2Frame is an unprotected MPEG-1 Layer II header at 128 kbit/s and 44.1 kHz.
	layer2Frame = fields{version: rawMPEG1, layer: rawLayer2, bitrate: 8, unprotected: true}
	// freeFrame is an unprotected free-format MPEG-1 Layer III header at 44.1 kHz.
	freeFrame = fields{version: rawMPEG1, layer: rawLayer3, unprotected: true}
)

// stream builds frames of the given headers, each filled with zeros to the size its header gives.
func stream(t *testing.T, heads ...fields) []byte {
	t.Helper()

	frames := make([][]byte, 0, len(heads))

	for _, f := range heads {
		head, ok := Parse(encode(f))
		require.True(t, ok)

		frames = append(frames, sized(f, head.Size()))
	}

	return slices.Concat(frames...)
}

// sized builds one frame of f's header, filled with zeros to size bytes.
func sized(f fields, size int) []byte {
	return append(encode(f), make([]byte, size-HeaderSize)...)
}

// TestFind covers confirmed frames, false syncs, the accept filter and windows that end too soon.
func TestFind(t *testing.T) {
	t.Parallel()

	two := stream(t, mpeg1Frame, mpeg1Frame)
	isLayer3 := func(h Header) bool { return h.Layer == Layer3 }

	tests := []struct {
		name      string
		window    []byte
		accept    func(Header) bool
		at        int
		confirmed bool
	}{
		{name: "frame at the start", window: two, at: 0, confirmed: true},
		{name: "junk before the frame", window: slices.Concat([]byte{0x00, 0x11, 0xff, 0x00}, two), at: 4, confirmed: true},
		{name: "false sync before the frame", window: slices.Concat(encode(mpeg1Frame), make([]byte, 6), two), at: 10, confirmed: true},
		{name: "stream changes at the next header", window: stream(t, mpeg1Frame, mpeg2Frame, mpeg2Frame), at: 417, confirmed: true},
		{name: "accept skips other frames", window: stream(t, layer2Frame, layer2Frame, mpeg1Frame, mpeg1Frame), accept: isLayer3, at: 834, confirmed: true},
		{name: "window ends before the next header", window: two[:419], at: 0},
		{name: "no frame", window: make([]byte, 16), at: 13},
		{name: "shorter than a header", window: []byte{0xff, 0xfb}, at: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			at, head, confirmed := Find(tt.window, tt.accept)

			assert.Equal(t, tt.at, at)
			assert.Equal(t, tt.confirmed, confirmed)

			if confirmed {
				want, _ := Parse(tt.window[at:])
				assert.Equal(t, want, head)
			}
		})
	}
}

// TestFindFreeFormat checks that Find measures free-format frames of each padding and slot size,
// and that the measured bitrate gives the frame sizes back.
func TestFindFreeFormat(t *testing.T) {
	t.Parallel()

	padded := freeFrame
	padded.padding = true

	layer1 := fields{version: rawMPEG1, layer: rawLayer1, unprotected: true}
	layer1Padded := layer1
	layer1Padded.padding = true

	tests := []struct {
		name   string
		window []byte
		size   int
	}{
		{"unpadded first", slices.Concat(sized(freeFrame, 300), sized(padded, 301), sized(freeFrame, 300)), 300},
		{"padded first", slices.Concat(sized(padded, 301), sized(freeFrame, 300), sized(padded, 301)), 301},
		{"Layer I slots", slices.Concat(sized(layer1, 96), sized(layer1Padded, 100), sized(layer1, 96)), 96},
		{"Layer I header off the slot grid", slices.Concat(offGrid(layer1), sized(layer1Padded, 100), sized(layer1, 96)), 96},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			at, head, confirmed := Find(tt.window, nil)

			require.True(t, confirmed)
			assert.Zero(t, at)
			assert.True(t, head.Free)
			assert.Equal(t, tt.size, head.Size())
		})
	}
}

// offGrid builds a 96-byte free-format Layer I frame of f's header that holds another such header
// 50 bytes in, which no frame spacing of whole 4-byte slots can reach.
func offGrid(f fields) []byte {
	frame := sized(f, 96)
	copy(frame[50:], encode(f))

	return frame
}

// TestFindFreeFormatUnconfirmed covers a free-format frame the window ends before confirming, one
// with no header after it and one with a second header but no third.
func TestFindFreeFormatUnconfirmed(t *testing.T) {
	t.Parallel()

	short := slices.Concat(sized(freeFrame, 300), sized(freeFrame, 300))
	at, _, confirmed := Find(short, nil)

	assert.Zero(t, at)
	assert.False(t, confirmed)

	lone := sized(freeFrame, 2*maxFreeSize)
	at, _, confirmed = Find(lone, nil)

	assert.Equal(t, len(lone)-HeaderSize+1, at)
	assert.False(t, confirmed)

	pair := slices.Concat(sized(freeFrame, 300), sized(freeFrame, 2*maxFreeSize))
	at, _, confirmed = Find(pair, nil)

	assert.Equal(t, len(pair)-HeaderSize+1, at)
	assert.False(t, confirmed)
}

// TestFindFile walks the frames of an encoded file and checks that each one confirms the next.
func TestFindFile(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/protected.mp3")
	require.NoError(t, err)

	at, first, confirmed := Find(data, nil)
	require.True(t, confirmed)
	require.Zero(t, at)

	frames := 0

	for at < len(data) {
		head, ok := Parse(data[at:])
		require.True(t, ok, "frame %d at %d", frames, at)
		require.True(t, first.SameStream(head))

		at += head.Size()
		frames++
	}

	assert.Equal(t, len(data), at)
	assert.Greater(t, frames, 2)
}
