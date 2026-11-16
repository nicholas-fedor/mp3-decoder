// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// Headers the sync tests build frames from.
var (
	// mpeg1Head is an MPEG-1 Layer III header at 128 kbit/s and 44.1 kHz, whose frames hold 417
	// bytes.
	mpeg1Head = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	// mpeg2Head is an MPEG-2 Layer III header at 64 kbit/s and 22.05 kHz, whose frames hold 208
	// bytes.
	mpeg2Head = mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8}
	// layer2Head is an MPEG-1 Layer II header at 128 kbit/s and 44.1 kHz, whose frames hold 417
	// bytes.
	layer2Head = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: 8}
	// freeHead is a free-format MPEG-1 Layer III header at 44.1 kHz.
	freeHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3}
	// layer1FreeHead is a free-format MPEG-1 Layer I header at 44.1 kHz.
	layer1FreeHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1}
)

// padded returns head with its padding bit set.
func padded(head mp3test.Header) mp3test.Header {
	head.Padding = true

	return head
}

// TestFind covers confirmed frames, false syncs, the accept filter and windows that end too soon.
func TestFind(t *testing.T) {
	t.Parallel()

	two := slices.Concat(mpeg1Head.Frame(417), mpeg1Head.Frame(417))
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
		{name: "false sync before the frame", window: slices.Concat(mpeg1Head.Frame(10), two), at: 10, confirmed: true},
		{
			name:   "stream changes at the next header",
			window: slices.Concat(mpeg1Head.Frame(417), mpeg2Head.Frame(208), mpeg2Head.Frame(208)),
			at:     417, confirmed: true,
		},
		{
			name:   "accept skips other frames",
			window: slices.Concat(layer2Head.Frame(417), layer2Head.Frame(417), two),
			accept: isLayer3, at: 834, confirmed: true,
		},
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

	tests := []struct {
		name   string
		window []byte
		size   int
	}{
		{"unpadded first", slices.Concat(freeHead.Frame(300), padded(freeHead).Frame(301), freeHead.Frame(300)), 300},
		{"padded first", slices.Concat(padded(freeHead).Frame(301), freeHead.Frame(300), padded(freeHead).Frame(301)), 301},
		{
			"Layer I slots",
			slices.Concat(layer1FreeHead.Frame(96), padded(layer1FreeHead).Frame(100), layer1FreeHead.Frame(96)), 96,
		},
		{
			"Layer I header off the slot grid",
			slices.Concat(offGrid(), padded(layer1FreeHead).Frame(100), layer1FreeHead.Frame(96)), 96,
		},
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

// offGrid builds a 96-byte free-format Layer I frame that holds another such header 50 bytes in,
// which no frame spacing of whole 4-byte slots can reach.
func offGrid() []byte {
	return mp3test.Place(layer1FreeHead.Frame(96), 50, layer1FreeHead.Bytes())
}

// TestFindFreeFormatUnconfirmed covers a free-format frame the window ends before confirming, one
// with no header after it and one with a second header but no third.
func TestFindFreeFormatUnconfirmed(t *testing.T) {
	t.Parallel()

	short := slices.Concat(freeHead.Frame(300), freeHead.Frame(300))
	at, _, confirmed := Find(short, nil)

	assert.Zero(t, at)
	assert.False(t, confirmed)

	lone := freeHead.Frame(2 * MaxSize)
	at, _, confirmed = Find(lone, nil)

	assert.Equal(t, len(lone)-HeaderSize+1, at)
	assert.False(t, confirmed)

	pair := slices.Concat(freeHead.Frame(300), freeHead.Frame(2*MaxSize))
	at, _, confirmed = Find(pair, nil)

	assert.Equal(t, len(pair)-HeaderSize+1, at)
	assert.False(t, confirmed)
}
