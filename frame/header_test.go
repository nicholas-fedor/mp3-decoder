// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestParse covers valid headers of every version and layer and each value Parse rejects.
func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head []byte
		want Header
		ok   bool
	}{
		{
			name: "MPEG-1 Layer III 128 kbit/s at 44.1 kHz",
			head: []byte{0xff, 0xfb, 0x90, 0x00},
			want: Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{
			name: "MPEG-2 Layer III protected, padded joint stereo",
			head: mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 9, Mode: mp3test.JointStereo, Extension: 2, Padding: true, Protected: true}.Bytes(),
			want: Header{
				SampleRate: 22050, Bitrate: 80000, Version: MPEG2, Layer: Layer3, Mode: JointStereo,
				Extension: 2, Padding: true, Protected: true,
			},
			ok: true,
		},
		{
			name: "MPEG-2.5 Layer III mono at 8 kHz",
			head: mp3test.Header{Version: mp3test.MPEG25, Layer: mp3test.Layer3, Bitrate: 1, Rate: 2, Mode: mp3test.Mono}.Bytes(),
			want: Header{SampleRate: 8000, Bitrate: 8000, Version: MPEG25, Layer: Layer3, Mode: Mono},
			ok:   true,
		},
		{
			name: "MPEG-2.5 at the top of the MPEG-2 bitrate table",
			head: mp3test.Header{Version: mp3test.MPEG25, Layer: mp3test.Layer3, Bitrate: 14, Rate: 1}.Bytes(),
			want: Header{SampleRate: 12000, Bitrate: 160000, Version: MPEG25, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{
			name: "MPEG-2.5 Layer II",
			head: mp3test.Header{Version: mp3test.MPEG25, Layer: mp3test.Layer2, Bitrate: 4}.Bytes(),
			want: Header{SampleRate: 11025, Bitrate: 32000, Version: MPEG25, Layer: Layer2, Mode: Stereo},
			ok:   true,
		},
		{
			name: "MPEG-1 Layer I at its highest bitrate",
			head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: 14, Rate: 1, Mode: mp3test.DualChannel}.Bytes(),
			want: Header{SampleRate: 48000, Bitrate: 448000, Version: MPEG1, Layer: Layer1, Mode: DualChannel},
			ok:   true,
		},
		{
			name: "free format",
			head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3}.Bytes(),
			want: Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3, Mode: Stereo, Free: true},
			ok:   true,
		},
		{
			name: "reserved emphasis",
			head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9, Emphasis: 2}.Bytes(),
			want: Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{name: "short", head: []byte{0xff, 0xfb, 0x90}},
		{name: "no sync word", head: []byte{0xff, 0xdb, 0x90, 0x00}},
		{name: "reserved version", head: mp3test.Header{Version: 1, Layer: mp3test.Layer3, Bitrate: 9}.Bytes()},
		{name: "reserved layer", head: mp3test.Header{Version: mp3test.MPEG1, Bitrate: 9}.Bytes()},
		{name: "forbidden bitrate", head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 15}.Bytes()},
		{name: "reserved sampling frequency", head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Rate: 3}.Bytes()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := Parse(tt.head)

			require.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestHeaderSize checks known frame sizes, including a padded MPEG-2 frame whose size an even
// halving of the MPEG-1 formula gets wrong.
func TestHeaderSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head Header
		want int
	}{
		{"MPEG-1 Layer III", Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3}, 417},
		{
			"MPEG-1 Layer III padded",
			Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Padding: true},
			418,
		},
		{
			"MPEG-2 Layer III padded",
			Header{SampleRate: 22050, Bitrate: 80000, Version: MPEG2, Layer: Layer3, Padding: true},
			262,
		},
		{"MPEG-1 Layer I", Header{SampleRate: 44100, Bitrate: 32000, Version: MPEG1, Layer: Layer1}, 32},
		{
			"MPEG-1 Layer I padded",
			Header{SampleRate: 44100, Bitrate: 32000, Version: MPEG1, Layer: Layer1, Padding: true},
			36,
		},
		{"MPEG-2 Layer II", Header{SampleRate: 24000, Bitrate: 160000, Version: MPEG2, Layer: Layer2}, 960},
		{"unmeasured free format", Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3, Free: true}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.head.Size())
		})
	}
}

// TestHeaderSizeClosedForm checks the size of every valid header against the per-layer formulas and
// against [MaxSize], and its samples against [MaxSamples].
func TestHeaderSizeClosedForm(t *testing.T) {
	t.Parallel()

	for _, version := range []uint32{mp3test.MPEG1, mp3test.MPEG2, mp3test.MPEG25} {
		for _, layer := range []uint32{mp3test.Layer1, mp3test.Layer2, mp3test.Layer3} {
			for index := uint32(1); index < bitrateForbidden; index++ {
				for rate := range uint32(rateReserved) {
					for _, padding := range []bool{false, true} {
						head, ok := Parse(mp3test.Header{
							Version: version, Layer: layer, Bitrate: index, Rate: rate, Padding: padding,
						}.Bytes())
						require.True(t, ok)

						assert.Equal(t, closedFormSize(head), head.Size(), "%+v", head)
						assert.LessOrEqual(t, head.Size(), MaxSize, "%+v", head)
						assert.LessOrEqual(t, head.Samples(), MaxSamples, "%+v", head)
					}
				}
			}
		}
	}
}

// closedFormSize computes a frame's size from the per-layer formulas.
//
// A Layer I frame holds 4 times (12 bitrate / rate + padding) bytes. An MPEG-2 or MPEG-2.5 Layer III
// frame holds 72 bitrate / rate + padding bytes. Every other frame holds 144 bitrate / rate + padding
// bytes.
func closedFormSize(h Header) int {
	pad := 0
	if h.Padding {
		pad = 1
	}

	switch {
	case h.Layer == Layer1:
		return (12*h.Bitrate/h.SampleRate + pad) * 4
	case h.Layer == Layer3 && h.Version != MPEG1:
		return 72*h.Bitrate/h.SampleRate + pad
	default:
		return 144*h.Bitrate/h.SampleRate + pad
	}
}

// TestHeaderSamples covers the frame length of each layer and version.
func TestHeaderSamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version Version
		layer   Layer
		want    int
	}{
		{MPEG1, Layer1, 384},
		{MPEG2, Layer1, 384},
		{MPEG25, Layer1, 384},
		{MPEG1, Layer2, 1152},
		{MPEG2, Layer2, 1152},
		{MPEG25, Layer2, 1152},
		{MPEG1, Layer3, 1152},
		{MPEG2, Layer3, 576},
		{MPEG25, Layer3, 576},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Header{Version: tt.version, Layer: tt.layer}.Samples(), "%d %d", tt.version, tt.layer)
	}
}

// TestHeaderChannels covers mono and the two-channel modes.
func TestHeaderChannels(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, Header{Mode: Mono}.Channels())

	for _, mode := range []Mode{Stereo, JointStereo, DualChannel} {
		assert.Equal(t, 2, Header{Mode: mode}.Channels())
	}
}

// TestHeaderSideInfoSize covers the Layer III side information sizes and the other layers, which
// have none.
func TestHeaderSideInfoSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		head Header
		want int
	}{
		{Header{Version: MPEG1, Layer: Layer3, Mode: Mono}, 17},
		{Header{Version: MPEG1, Layer: Layer3, Mode: JointStereo}, 32},
		{Header{Version: MPEG2, Layer: Layer3, Mode: Mono}, 9},
		{Header{Version: MPEG25, Layer: Layer3, Mode: Stereo}, 17},
		{Header{Version: MPEG1, Layer: Layer2, Mode: Stereo}, 0},
		{Header{Version: MPEG1, Layer: Layer1, Mode: Mono}, 0},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.head.SideInfoSize(), "%+v", tt.head)
	}
}

// TestHeaderDataAt checks that the data offset adds the CRC to the header and side information.
func TestHeaderDataAt(t *testing.T) {
	t.Parallel()

	head := Header{Version: MPEG1, Layer: Layer3, Mode: Stereo}
	assert.Equal(t, 36, head.DataAt())

	head.Protected = true
	assert.Equal(t, 38, head.DataAt())

	assert.Equal(t, 6, Header{Version: MPEG1, Layer: Layer2, Protected: true}.DataAt())
}

// TestHeaderMinDataSize checks the smallest frame of a stream against its lowest bitrate.
func TestHeaderMinDataSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head Header
		want int
	}{
		// 1152*32000/(8*48000) = 96 bytes, less 4 + 2 + 32.
		{"MPEG-1", Header{SampleRate: 48000, Bitrate: 320000, Version: MPEG1, Layer: Layer3, Mode: Mono}, 58},
		// 576*8000/(8*24000) = 24 bytes, less 4 + 2 + 17.
		{"MPEG-2", Header{SampleRate: 24000, Bitrate: 64000, Version: MPEG2, Layer: Layer3, Padding: true}, 1},
		// 576*8000/(8*8000) = 72 bytes, less 4 + 2 + 17.
		{"MPEG-2.5", Header{SampleRate: 8000, Bitrate: 8000, Version: MPEG25, Layer: Layer3}, 49},
		// 1152*100000/(8*44100) = 326 bytes, less 4 + 2 + 32.
		{"free format", Header{SampleRate: 44100, Bitrate: 100000, Version: MPEG1, Layer: Layer3, Free: true}, 288},
		{"free format unmeasured", Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3, Free: true}, 0},
		// 1152*32000/(8*48000) = 96 bytes, less 4 + 2.
		{"Layer II", Header{SampleRate: 48000, Bitrate: 192000, Version: MPEG1, Layer: Layer2}, 90},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.head.minDataSize(), tt.name)
	}
}

// TestHeaderPreroll checks the frames that refill the bit reservoir, then those that refill the
// filterbanks, from the smallest frame of each stream.
func TestHeaderPreroll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head Header
		want int
	}{
		// 511 bytes in frames of 104 - 38 bytes, then one frame of two granules.
		{"MPEG-1 at 44.1 kHz", Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3}, 8 + 1},
		// 511 bytes in frames of 96 - 38 bytes.
		{"MPEG-1 at 48 kHz", Header{SampleRate: 48000, Version: MPEG1, Layer: Layer3}, 9 + 1},
		// 255 bytes in frames of 24 - 23 bytes, then two frames of one granule.
		{"MPEG-2 at 24 kHz", Header{SampleRate: 24000, Version: MPEG2, Layer: Layer3}, 255 + 2},
		// 255 bytes in frames of 72 - 23 bytes.
		{"MPEG-2.5 at 8 kHz", Header{SampleRate: 8000, Version: MPEG25, Layer: Layer3}, 6 + 2},
		// 511 bytes in frames of 326 - 38 bytes.
		{"free format", Header{SampleRate: 44100, Bitrate: 100000, Version: MPEG1, Layer: Layer3, Free: true}, 2 + 1},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.head.Preroll(), tt.name)
	}
}

// TestHeaderSameStream checks that a stream may change its bitrate, padding and mode but not its
// version, layer, sampling frequency or free format.
func TestHeaderSameStream(t *testing.T) {
	t.Parallel()

	base := Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Mode: JointStereo}

	same := []Header{
		{SampleRate: 44100, Bitrate: 320000, Version: MPEG1, Layer: Layer3, Mode: JointStereo},
		{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Mode: Mono, Padding: true},
	}
	for _, other := range same {
		assert.True(t, base.SameStream(other), "%+v", other)
	}

	differ := []Header{
		{SampleRate: 48000, Bitrate: 128000, Version: MPEG1, Layer: Layer3},
		{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer2},
		{SampleRate: 22050, Bitrate: 128000, Version: MPEG2, Layer: Layer3},
		{SampleRate: 44100, Version: MPEG1, Layer: Layer3, Free: true},
	}
	for _, other := range differ {
		assert.False(t, base.SameStream(other), "%+v", other)
	}
}
