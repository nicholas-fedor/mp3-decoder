// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestDecodeEncodedFiles decodes every frame of the MPEG-1 files LAME encoded.
//
// Every sample must be finite and within twice full scale, and a mono file must give each sample
// to both channels.
func TestDecodeEncodedFiles(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var decoder layer3.Decoder

			pcm := make([][2]float64, frame.MaxSamples)
			loud := 0

			for at, data := range encodedFrames(t, name) {
				head, _ := frame.Parse(data)
				require.NoError(t, decoder.Decode(pcm, head, data), "frame %d", at)

				for sample, pair := range pcm {
					for _, value := range pair {
						require.False(t, math.IsNaN(value) || math.Abs(value) > 2, "frame %d sample %d", at, sample)
					}

					if head.Channels() == 1 {
						require.InDelta(t, pair[0], pair[1], 0, "frame %d sample %d", at, sample)
					}

					if math.Abs(pair[0]) > 0.1 {
						loud++
					}
				}
			}

			assert.Positive(t, loud, "samples above -20 dB")
		})
	}
}

// TestDecodeJoined decodes a file from a frame whose main data begins in the frames before it, as
// on joining a stream midway.
//
// The granules whose main data is missing must be silent, and the frames after them must sound.
func TestDecodeJoined(t *testing.T) {
	t.Parallel()

	frames := encodedFrames(t, "mpeg1-joint.mp3")

	var side layer3.SideInfo

	join := 0

	for at, data := range frames[2:] {
		head, _ := frame.Parse(data)
		require.NoError(t, side.Parse(head, data))

		if side.MainDataBegin > 0 {
			join = at + 2

			break
		}
	}

	require.Positive(t, join, "a frame whose main data begins earlier")

	var decoder layer3.Decoder

	pcm := make([][2]float64, frame.MaxSamples)
	head, _ := frame.Parse(frames[join])

	require.NoError(t, decoder.Decode(pcm, head, frames[join]))
	assert.Equal(t, make([][2]float64, 576), pcm[:576], "first granule")

	for _, data := range frames[join+1 : join+4] {
		head, _ = frame.Parse(data)
		require.NoError(t, decoder.Decode(pcm, head, data))
	}

	assert.NotEqual(t, make([][2]float64, frame.MaxSamples), pcm, "a later frame")
}

// TestDecodeErrors checks that a frame the decoder refuses leaves the samples as they were.
func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	lsf := mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8}
	mpeg1 := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	broken := mp3test.SideInfo{Granules: [2][2]mp3test.Granule{{{WindowSwitching: true}}}}

	tests := []struct {
		want error
		name string
		head mp3test.Header
		side mp3test.SideInfo
	}{
		{layer3.ErrVersion, "MPEG-2", lsf, mp3test.SideInfo{}},
		{layer3.ErrBlockType, "window switching with block type 0", mpeg1, broken},
	}

	for _, tt := range tests {
		data, head := frameWith(t, tt.head, &tt.side)

		pcm := make([][2]float64, frame.MaxSamples)
		pcm[7] = [2]float64{0.5, -0.5}

		var decoder layer3.Decoder

		require.ErrorIs(t, decoder.Decode(pcm, head, data), tt.want, tt.name)
		assert.Equal(t, [2]float64{0.5, -0.5}, pcm[7], tt.name)
	}

	data, head := frameWith(t, mpeg1, &mp3test.SideInfo{})

	var decoder layer3.Decoder

	require.ErrorIs(t, decoder.Decode(make([][2]float64, frame.MaxSamples), head, data[:20]), layer3.ErrShortFrame)
}
