// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// Headers of MPEG-1 Layer III frames at 128 kbit/s and 44.1 kHz, which hold 417 bytes.
var (
	// plainHead is an unprotected stereo header.
	plainHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	// protectedHead is the same header with a CRC.
	protectedHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9, Protected: true}
)

// tagged builds a 417-byte frame of head with payload at an offset.
func tagged(head mp3test.Header, at int, payload []byte) []byte {
	return mp3test.Place(head.Frame(417), at, payload)
}

// TestParseInfo covers tags after the CRC, tags without some fields or the LAME extension, tags
// cut short, VBRI tags and frames without a tag, among them Layer I and II frames whose bytes read
// as a tag.
func TestParseInfo(t *testing.T) {
	t.Parallel()

	toc := make([]byte, tocSize)

	tests := []struct {
		name string
		data []byte
		want Info
		ok   bool
	}{
		{
			name: "Info tag after the CRC",
			data: tagged(protectedHead, 38, mp3test.Tag{Magic: infoMagic, Flags: mp3test.HasFrames, Frames: 7}.Bytes()),
			want: Info{Frames: 7},
			ok:   true,
		},
		{
			name: "frame count only",
			data: tagged(plainHead, 36, mp3test.Tag{Magic: xingMagic, Flags: mp3test.HasFrames, Frames: 7}.Bytes()),
			want: Info{Frames: 7},
			ok:   true,
		},
		{
			name: "every field and the LAME extension",
			data: tagged(plainHead, 36, mp3test.Tag{
				Magic: xingMagic, Flags: mp3test.HasAll, Frames: 7, Size: 4000, TOC: toc, Quality: 50,
				Encoder: "LAME3.100", Delay: 1105, Padding: 600,
			}.Bytes()),
			want: Info{TOC: toc, Frames: 7, Bytes: 4000, Delay: 1105, Padding: 600, Gapless: true},
			ok:   true,
		},
		{
			name: "another encoder's extension",
			data: tagged(plainHead, 36, mp3test.Tag{
				Magic: infoMagic, Flags: mp3test.HasFrames, Frames: 7, Encoder: "Lavc61.3", Delay: 576, Padding: 1,
			}.Bytes()),
			want: Info{Frames: 7, Delay: 576, Padding: 1, Gapless: true},
			ok:   true,
		},
		{
			name: "no extension",
			data: tagged(plainHead, 36, mp3test.Tag{
				Magic: xingMagic, Flags: mp3test.HasFrames | mp3test.HasBytes, Frames: 7, Size: 4000,
			}.Bytes()),
			want: Info{Frames: 7, Bytes: 4000},
			ok:   true,
		},
		{
			name: "frame ends before the table",
			data: tagged(plainHead, 36, mp3test.Tag{
				Magic: xingMagic, Flags: mp3test.HasFrames | mp3test.HasTOC, Frames: 7,
			}.Bytes())[:60],
			want: Info{Frames: 7},
			ok:   true,
		},
		{
			name: "frame ends before the frame count",
			data: tagged(plainHead, 36, mp3test.Tag{Magic: xingMagic, Flags: mp3test.HasFrames}.Bytes())[:46],
			ok:   true,
		},
		{
			name: "VBRI tag",
			data: tagged(plainHead, mp3test.VBRIAt, mp3test.VBRI{Version: vbriVersion, Size: 5000, Frames: 9}.Bytes()),
			want: Info{Frames: 9, Bytes: 5000},
			ok:   true,
		},
		{
			name: "VBRI tag of another version",
			data: tagged(plainHead, mp3test.VBRIAt, mp3test.VBRI{Version: 2, Size: 5000, Frames: 9}.Bytes()),
		},
		{name: "no tag", data: tagged(plainHead, 36, mp3test.Tag{Magic: "Nope"}.Bytes())},
		{
			name: "a Layer I frame whose bytes read as an Info tag",
			data: tagged(mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: 9}, frame.HeaderSize,
				mp3test.Tag{Magic: infoMagic, Flags: mp3test.HasFrames, Frames: 7}.Bytes()),
		},
		{
			name: "a Layer II frame whose bytes read as a VBRI tag",
			data: tagged(mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: 9}, mp3test.VBRIAt,
				mp3test.VBRI{Version: vbriVersion, Size: 5000, Frames: 9}.Bytes()),
		},
		{name: "frame ends before the flags", data: tagged(plainHead, 36, mp3test.Tag{Magic: xingMagic}.Bytes())[:42]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			head, ok := frame.Parse(tt.data)
			require.True(t, ok)

			info, ok := ParseInfo(head, tt.data)

			require.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, info)
		})
	}
}

// TestInfoSamples covers a tag without a frame count and delays longer than the stream.
func TestInfoSamples(t *testing.T) {
	t.Parallel()

	head, ok := frame.Parse(plainHead.Bytes())
	require.True(t, ok)

	assert.Zero(t, Info{Delay: 576, Gapless: true}.Samples(head))
	assert.Equal(t, int64(2304), Info{Frames: 2}.Samples(head))
	assert.Zero(t, Info{Frames: 1, Delay: 1000, Padding: 1000, Gapless: true}.Samples(head))
}

// TestInfoOffsetAt checks interpolation between the seek table's entries, the clamp at either end
// and the cases without a table or a size.
//
// Offsets may differ by one byte with the rounding of the interpolation.
func TestInfoOffsetAt(t *testing.T) {
	t.Parallel()

	toc := make([]byte, tocSize)
	for i := range toc {
		toc[i] = byte(i * 2)
	}

	info := Info{TOC: toc}

	tests := []struct {
		name     string
		fraction float64
		want     int64
	}{
		{"start", 0, 0},
		{"before the start", -1, 0},
		{"on an entry", 0.1, 20 * 25600 / 256},
		{"between entries", 0.105, 21 * 25600 / 256},
		{"clamped short of the end", 1, 25542},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := info.OffsetAt(tt.fraction, 25600)

			require.True(t, ok)
			assert.InDelta(t, tt.want, got, 1)
		})
	}

	_, ok := Info{}.OffsetAt(0.5, 25600)
	assert.False(t, ok)

	_, ok = info.OffsetAt(0.5, 0)
	assert.False(t, ok)
}
