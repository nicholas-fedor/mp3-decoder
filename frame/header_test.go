// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fields holds the raw bit values of a frame header's fields.
type fields struct {
	version, layer, bitrate, rate, mode, ext, emphasis uint32
	padding, unprotected                               bool
}

// Raw field values for building headers.
const (
	// rawMPEG1 is the version bits of MPEG-1.
	rawMPEG1 = 3
	// rawMPEG2 is the version bits of MPEG-2.
	rawMPEG2 = 2
	// rawMPEG25 is the version bits of MPEG-2.5.
	rawMPEG25 = 0
	// rawLayer1 is the layer bits of Layer I.
	rawLayer1 = 3
	// rawLayer2 is the layer bits of Layer II.
	rawLayer2 = 2
	// rawLayer3 is the layer bits of Layer III.
	rawLayer3 = 1
)

// encode builds the four header bytes for f.
func encode(f fields) []byte {
	word := uint32(syncWord)<<syncShift | f.version<<versionShift | f.layer<<layerShift |
		f.bitrate<<bitrateShift | f.rate<<rateShift | f.mode<<modeShift | f.ext<<extensionShift | f.emphasis

	if f.unprotected {
		word |= 1 << protectionShift
	}

	if f.padding {
		word |= 1 << paddingShift
	}

	return binary.BigEndian.AppendUint32(nil, word)
}

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
			head: encode(fields{version: rawMPEG2, layer: rawLayer3, bitrate: 9, mode: 1, ext: 2, padding: true}),
			want: Header{
				SampleRate: 22050, Bitrate: 80000, Version: MPEG2, Layer: Layer3, Mode: JointStereo,
				Extension: 2, Padding: true, Protected: true,
			},
			ok: true,
		},
		{
			name: "MPEG-2.5 Layer III mono at 8 kHz",
			head: encode(fields{version: rawMPEG25, layer: rawLayer3, bitrate: 1, rate: 2, mode: 3, unprotected: true}),
			want: Header{SampleRate: 8000, Bitrate: 8000, Version: MPEG25, Layer: Layer3, Mode: Mono},
			ok:   true,
		},
		{
			name: "MPEG-2.5 at the top of the MPEG-2 bitrate table",
			head: encode(fields{version: rawMPEG25, layer: rawLayer3, bitrate: 14, rate: 1, unprotected: true}),
			want: Header{SampleRate: 12000, Bitrate: 160000, Version: MPEG25, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{
			name: "MPEG-2.5 Layer II",
			head: encode(fields{version: rawMPEG25, layer: rawLayer2, bitrate: 4, unprotected: true}),
			want: Header{SampleRate: 11025, Bitrate: 32000, Version: MPEG25, Layer: Layer2, Mode: Stereo},
			ok:   true,
		},
		{
			name: "MPEG-1 Layer I at its highest bitrate",
			head: encode(fields{version: rawMPEG1, layer: rawLayer1, bitrate: 14, rate: 1, mode: 2, unprotected: true}),
			want: Header{SampleRate: 48000, Bitrate: 448000, Version: MPEG1, Layer: Layer1, Mode: DualChannel},
			ok:   true,
		},
		{
			name: "free format",
			head: encode(fields{version: rawMPEG1, layer: rawLayer3, unprotected: true}),
			want: Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{
			name: "reserved emphasis",
			head: encode(fields{version: rawMPEG1, layer: rawLayer3, bitrate: 9, emphasis: 2, unprotected: true}),
			want: Header{SampleRate: 44100, Bitrate: 128000, Version: MPEG1, Layer: Layer3, Mode: Stereo},
			ok:   true,
		},
		{name: "short", head: []byte{0xff, 0xfb, 0x90}},
		{name: "no sync word", head: []byte{0xff, 0xdb, 0x90, 0x00}},
		{name: "reserved version", head: encode(fields{version: 1, layer: rawLayer3, bitrate: 9})},
		{name: "reserved layer", head: encode(fields{version: rawMPEG1, bitrate: 9})},
		{name: "forbidden bitrate", head: encode(fields{version: rawMPEG1, layer: rawLayer3, bitrate: 15})},
		{name: "reserved sampling frequency", head: encode(fields{version: rawMPEG1, layer: rawLayer3, rate: 3})},
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
		{"free format", Header{SampleRate: 44100, Version: MPEG1, Layer: Layer3}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.head.Size())
		})
	}
}

// TestHeaderSizeClosedForm checks the size of every valid header against the per-layer formulas.
func TestHeaderSizeClosedForm(t *testing.T) {
	t.Parallel()

	for _, version := range []uint32{rawMPEG1, rawMPEG2, rawMPEG25} {
		for _, layer := range []uint32{rawLayer1, rawLayer2, rawLayer3} {
			for index := uint32(1); index < bitrateForbidden; index++ {
				for rate := range uint32(rateReserved) {
					for _, padding := range []bool{false, true} {
						head, ok := Parse(encode(fields{
							version: version, layer: layer, bitrate: index, rate: rate, padding: padding,
						}))
						require.True(t, ok)

						assert.Equal(t, closedFormSize(head), head.Size(), "%+v", head)
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
		{SampleRate: 44100, Version: MPEG1, Layer: Layer3},
	}
	for _, other := range differ {
		assert.False(t, base.SameStream(other), "%+v", other)
	}
}
