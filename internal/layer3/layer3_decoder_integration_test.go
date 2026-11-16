// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer3_test

import (
	"math"
	"slices"
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
				decode(t, &decoder, pcm, head, data)

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
// The frame whose main data is missing must be silent in both granules, and the frames after it
// must sound.
func TestDecodeJoined(t *testing.T) {
	t.Parallel()

	frames := encodedFrames(t, "mpeg1-joint.mp3")
	join := reachingBack(t, frames)

	var decoder layer3.Decoder

	pcm := make([][2]float64, frame.MaxSamples)
	head, _ := frame.Parse(frames[join])

	assert.False(t, decode(t, &decoder, pcm, head, frames[join]), "the joined frame's main data is all there")
	assert.True(t, slices.Equal(make([][2]float64, frame.MaxSamples), pcm), "the joined frame sounds")

	sounds := false

	for _, data := range frames[join+1 : join+8] {
		head, _ = frame.Parse(data)
		decode(t, &decoder, pcm, head, data)

		sounds = sounds || !slices.Equal(make([][2]float64, frame.MaxSamples), pcm)
	}

	assert.True(t, sounds, "no later frame sounds")
}

// TestDecodeResync checks that a resync drops the main data of the frames before it, while the
// filterbanks carry on.
//
// The frame after the resync reaches back to main data the decoder no longer holds, so it decodes
// otherwise than without the resync. The frames after it reach back only to main data since the
// resync, so the output soon matches a decoder that never resynced, sample for sample.
func TestDecodeResync(t *testing.T) {
	t.Parallel()

	frames := encodedFrames(t, "mpeg1-joint.mp3")
	join := reachingBack(t, frames)

	var resynced, steady layer3.Decoder

	got := make([][2]float64, frame.MaxSamples)
	want := make([][2]float64, frame.MaxSamples)

	for at, data := range frames[:join+8] {
		if at == join {
			resynced.Resync()
		}

		head, _ := frame.Parse(data)
		decode(t, &resynced, got, head, data)
		decode(t, &steady, want, head, data)

		if at == join {
			assert.NotEqual(t, want, got, "the frame after the resync decodes as without it")
		}
	}

	assert.Equal(t, want, got, "the output has not converged 8 frames after the resync")
}

// TestDecodePreroll checks that a decoder reset and started as many frames before a frame as
// frame.Header.Preroll gives decodes that frame bit for bit as the whole stream does.
//
// The decoder is reset after decoding other frames of the stream, so a Reset that kept any state
// would change the frames that start the stream.
func TestDecodePreroll(t *testing.T) {
	t.Parallel()

	for _, name := range encodedFiles {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			frames := encodedFrames(t, name)
			first, _ := frame.Parse(frames[0])
			preroll := first.Preroll()
			samples := first.Samples()

			var steady, started layer3.Decoder

			whole := make([][2]float64, len(frames)*samples)

			for at, data := range frames {
				head, _ := frame.Parse(data)
				decode(t, &steady, whole[at*samples:], head, data)
			}

			got := make([][2]float64, samples)

			for target := range frames {
				started.Reset()

				for _, data := range frames[max(target-preroll, 0) : target+1] {
					head, _ := frame.Parse(data)
					decode(t, &started, got, head, data)
				}

				if !slices.Equal(whole[target*samples:(target+1)*samples], got) {
					require.Failf(t, "frame decodes otherwise", "frame %d with %d frames before it", target, preroll)
				}
			}
		})
	}
}

// TestDecodeChannelChange checks that the frames after a change between mono and stereo decode as
// a fresh decoder decodes them, since both channels' filterbanks start afresh.
//
// Each file's frames reach back only to its own frames, so the bit reservoir gives the same main
// data either way.
func TestDecodeChannelChange(t *testing.T) {
	t.Parallel()

	for _, files := range [][2]string{{"mpeg1-mono.mp3", "mpeg1-joint.mp3"}, {"mpeg1-joint.mp3", "mpeg1-mono.mp3"}} {
		t.Run(files[0]+" then "+files[1], func(t *testing.T) {
			t.Parallel()

			var changed, fresh layer3.Decoder

			got := make([][2]float64, frame.MaxSamples)
			want := make([][2]float64, frame.MaxSamples)

			for _, data := range encodedFrames(t, files[0])[:3] {
				head, _ := frame.Parse(data)
				decode(t, &changed, got, head, data)
			}

			for at, data := range encodedFrames(t, files[1])[:4] {
				head, _ := frame.Parse(data)
				decode(t, &changed, got, head, data)
				decode(t, &fresh, want, head, data)
				assert.True(t, slices.Equal(want, got), "frame %d", at)
			}
		})
	}
}

// TestDecodeMissingAfterMixed checks that a joint stereo granule of short blocks whose main data
// is missing gives silence after a granule of mixed blocks.
//
// The missing granule reads no scalefactors, so the right channel's still hold the mixed granule's.
func TestDecodeMissingAfterMixed(t *testing.T) {
	t.Parallel()

	var mixed, short mp3test.SideInfo

	mixed.Granules[0][1] = mp3test.Granule{WindowSwitching: true, BlockType: 2, MixedBlock: true}
	short.MainDataBegin = 255
	short.Granules[0][1] = mp3test.Granule{WindowSwitching: true, BlockType: 2}

	first, firstHead := frameWith(t, mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8}, &mixed)
	second, secondHead := frameWith(t, mp3test.Header{
		Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8, Mode: mp3test.JointStereo, Extension: 1,
	}, &short)

	var decoder layer3.Decoder

	pcm := make([][2]float64, frame.MaxSamples)

	decode(t, &decoder, pcm, firstHead, first)
	assert.False(t, decode(t, &decoder, pcm, secondHead, second))
	assert.Equal(t, make([][2]float64, secondHead.Samples()), pcm[:secondHead.Samples()])
}

// TestDecodeCRC checks that a frame whose CRC does not match its side information decodes all the
// same, to the samples of the frame with its CRC intact.
func TestDecodeCRC(t *testing.T) {
	t.Parallel()

	data := encodedFrames(t, "protected.mp3")[0]
	head, _ := frame.Parse(data)
	broken := slices.Clone(data)
	broken[frame.HeaderSize] ^= 1

	var intact, damaged layer3.Decoder

	want := make([][2]float64, frame.MaxSamples)
	got := make([][2]float64, frame.MaxSamples)

	decode(t, &intact, want, head, data)

	matched, whole, err := damaged.Decode(got, head, broken)
	require.NoError(t, err)
	assert.False(t, matched)
	assert.True(t, whole)
	assert.Equal(t, want, got)
}

// TestDecodeErrors checks that a frame the decoder refuses leaves the samples as they were.
func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	mpeg1 := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	broken := mp3test.SideInfo{Granules: [2][2]mp3test.Granule{{{WindowSwitching: true}}}}

	tests := []struct {
		want error
		name string
		head mp3test.Header
		side mp3test.SideInfo
	}{
		{layer3.ErrBlockType, "window switching with block type 0", mpeg1, broken},
	}

	for _, tt := range tests {
		data, head := frameWith(t, tt.head, &tt.side)

		pcm := make([][2]float64, frame.MaxSamples)
		pcm[7] = [2]float64{0.5, -0.5}

		var decoder layer3.Decoder

		_, _, err := decoder.Decode(pcm, head, data)
		require.ErrorIs(t, err, tt.want, tt.name)
		assert.Equal(t, [2]float64{0.5, -0.5}, pcm[7], tt.name)
	}

	data, head := frameWith(t, mpeg1, &mp3test.SideInfo{})

	var decoder layer3.Decoder

	_, _, err := decoder.Decode(make([][2]float64, frame.MaxSamples), head, data[:20])
	require.ErrorIs(t, err, layer3.ErrShortFrame)
}

// reachingBack returns the first frame after the first two whose main data begins in the frames
// before it.
func reachingBack(t *testing.T, frames [][]byte) int {
	t.Helper()

	var side layer3.SideInfo

	for at, data := range frames[2:] {
		head, _ := frame.Parse(data)
		require.NoError(t, side.Parse(head, data))

		if side.MainDataBegin > 0 {
			return at + 2
		}
	}

	require.Fail(t, "no frame's main data begins earlier")

	return 0
}

// decode decodes a frame that must decode with any CRC matching, and reports whether its main
// data was all there.
func decode(tb testing.TB, decoder *layer3.Decoder, pcm [][2]float64, head frame.Header, data []byte) bool {
	tb.Helper()

	intact, whole, err := decoder.Decode(pcm, head, data)
	require.NoError(tb, err)
	require.True(tb, intact, "CRC")

	return whole
}
