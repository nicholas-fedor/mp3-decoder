// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer12"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// samples is the number of samples a Layer I frame holds for each channel.
const samples = mp3test.Layer1Sets * mp3test.Subbands

// TestDecode checks frames of every mode, of MPEG-1 and MPEG-2, with and without a CRC, against
// the requantization formula and the synthesis filterbank, sample by sample.
//
// Each test decodes three frames in turn, so the filterbanks carry each frame into the next. A
// mono frame gives each sample to both channels.
func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head mp3test.Header
	}{
		{"mono", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Mono}},
		{"stereo", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Stereo}},
		{"dual channel", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.DualChannel}},
		{"joint stereo from subband 4", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo}},
		{"joint stereo from subband 8", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo, Extension: 1}},
		{"joint stereo from subband 12", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo, Extension: 2}},
		{"joint stereo from subband 16", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo, Extension: 3}},
		{"MPEG-2 with a CRC", mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer1, Mode: mp3test.JointStereo, Protected: true}},
		{"mono with a CRC", mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Mono, Protected: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				decoder layer12.Decoder
				banks   [2]synth.Filterbank
			)

			random := rand.New(rand.NewPCG(1, 2))
			got := make([][2]float64, frame.MaxSamples)

			for at := range 3 {
				fields := mp3test.RandomLayer1Audio(random, 14)
				data, head := layer1Frame(t, tt.head, &fields)

				intact, err := decoder.Decode(got, head, data)
				require.NoError(t, err)
				assert.True(t, intact, "CRC")

				want := reference(tt.head, &fields, &banks)

				for sample := range samples {
					require.InDelta(t, want[sample][0], got[sample][0], 1e-12, "frame %d sample %d", at, sample)
					require.InDelta(t, want[sample][1], got[sample][1], 1e-12, "frame %d sample %d", at, sample)
				}
			}
		})
	}
}

// TestDecodeCRC checks that a frame whose CRC does not match its bit allocation decodes all the
// same, to the samples of the frame with its CRC intact.
func TestDecodeCRC(t *testing.T) {
	t.Parallel()

	fields := mp3test.RandomLayer1Audio(rand.New(rand.NewPCG(3, 4)), 14)
	data, head := layer1Frame(t, mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Protected: true}, &fields)

	broken := slices.Clone(data)
	broken[frame.HeaderSize] ^= 1

	var intact, damaged layer12.Decoder

	want := make([][2]float64, frame.MaxSamples)
	got := make([][2]float64, frame.MaxSamples)

	matched, err := intact.Decode(want, head, data)
	require.NoError(t, err)
	require.True(t, matched)

	matched, err = damaged.Decode(got, head, broken)
	require.NoError(t, err)
	assert.False(t, matched)
	assert.Equal(t, want, got)
}

// TestDecodeForbiddenAllocation checks that a frame with the forbidden allocation in any subband
// fails and leaves the samples as they were, while its CRC is still checked.
func TestDecodeForbiddenAllocation(t *testing.T) {
	t.Parallel()

	for _, sb := range []int{0, 4, 31} {
		var fields mp3test.Layer1Audio

		fields.Allocation[0][sb] = 15
		data, head := layer1Frame(t, mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo, Protected: true}, &fields)

		pcm := make([][2]float64, frame.MaxSamples)
		pcm[7] = [2]float64{0.5, -0.5}

		var decoder layer12.Decoder

		intact, err := decoder.Decode(pcm, head, data)
		require.ErrorIs(t, err, layer12.ErrAllocation, "subband %d", sb)
		assert.True(t, intact, "subband %d", sb)
		assert.Equal(t, [2]float64{0.5, -0.5}, pcm[7], "subband %d", sb)
	}
}

// TestDecodeUnusedScalefactor checks that the scalefactor index no encoder writes silences its
// subband.
func TestDecodeUnusedScalefactor(t *testing.T) {
	t.Parallel()

	fields := mp3test.RandomLayer1Audio(rand.New(rand.NewPCG(5, 6)), 14)

	for sb := range mp3test.Subbands {
		fields.Allocation[0][sb] = max(fields.Allocation[0][sb], 1)
		fields.Scalefactors[0][sb] = 63
	}

	data, head := layer1Frame(t, mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Mono}, &fields)
	pcm := make([][2]float64, frame.MaxSamples)

	var decoder layer12.Decoder

	_, err := decoder.Decode(pcm, head, data)
	require.NoError(t, err)
	assert.Equal(t, make([][2]float64, samples), pcm[:samples])
}

// TestDecodeChannelChange checks that the frames after a change between mono and stereo decode as
// a fresh decoder decodes them, since both channels' filterbanks start afresh.
func TestDecodeChannelChange(t *testing.T) {
	t.Parallel()

	mono := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Mono}
	stereo := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Stereo}

	for _, heads := range [][2]mp3test.Header{{mono, stereo}, {stereo, mono}} {
		random := rand.New(rand.NewPCG(7, 8))

		var changed, fresh layer12.Decoder

		got := make([][2]float64, frame.MaxSamples)
		want := make([][2]float64, frame.MaxSamples)

		fields := mp3test.RandomLayer1Audio(random, 14)
		data, head := layer1Frame(t, heads[0], &fields)
		_, err := changed.Decode(got, head, data)
		require.NoError(t, err)

		for at := range 2 {
			fields = mp3test.RandomLayer1Audio(random, 14)
			data, head = layer1Frame(t, heads[1], &fields)

			_, err = changed.Decode(got, head, data)
			require.NoError(t, err)
			_, err = fresh.Decode(want, head, data)
			require.NoError(t, err)
			assert.Equal(t, want, got, "frame %d after the change", at)
		}
	}
}

// TestReset checks that a reset decoder decodes as a fresh one.
func TestReset(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewPCG(9, 10))
	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Stereo}

	var used, fresh layer12.Decoder

	got := make([][2]float64, frame.MaxSamples)
	want := make([][2]float64, frame.MaxSamples)

	fields := mp3test.RandomLayer1Audio(random, 14)
	data, parsed := layer1Frame(t, head, &fields)
	_, err := used.Decode(got, parsed, data)
	require.NoError(t, err)

	used.Reset()

	fields = mp3test.RandomLayer1Audio(random, 14)
	data, parsed = layer1Frame(t, head, &fields)
	_, err = used.Decode(got, parsed, data)
	require.NoError(t, err)
	_, err = fresh.Decode(want, parsed, data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// layer1Frame builds a frame of a Layer I header at 448 kbit/s holding audio data, with its CRC
// when the header is protected.
//
// The frame holds the audio data whole, even where it is larger than a frame of the bitrate.
func layer1Frame(tb testing.TB, head mp3test.Header, fields *mp3test.Layer1Audio) ([]byte, frame.Header) {
	tb.Helper()

	head.Bitrate = 14

	parsed, ok := frame.Parse(head.Bytes())
	require.True(tb, ok)

	audio := head.Layer1Audio(fields)
	data := mp3test.Place(head.Frame(max(parsed.Size(), parsed.DataAt()+len(audio))), parsed.DataAt(), audio)

	if head.Protected {
		mp3test.Protect(data, mp3test.Layer1AllocationBits*allocations(head))
	}

	return data, parsed
}

// allocations returns the number of allocations a Layer I frame of a header carries, which is one
// for each subband and channel below the bound and one for each subband from it on.
func allocations(head mp3test.Header) int {
	switch head.Mode {
	case mp3test.Mono:
		return mp3test.Subbands
	case mp3test.JointStereo:
		bound := 4 * (int(head.Extension) + 1)

		return 2*bound + mp3test.Subbands - bound
	default:
		return 2 * mp3test.Subbands
	}
}

// reference decodes a Layer I frame by the requantization formula and the synthesis filterbank.
//
// A sample of n bits with code c and scalefactor index i is 2^(1-i/3) times 2(c+1-2^(n-1))/(2^n-1).
// From the bound on, both channels take the first channel's allocation and code, each with its own
// scalefactor.
func reference(head mp3test.Header, fields *mp3test.Layer1Audio, banks *[2]synth.Filterbank) [][2]float64 {
	channels, bound := 2, mp3test.Subbands

	switch head.Mode {
	case mp3test.Mono:
		channels = 1
	case mp3test.JointStereo:
		bound = 4 * (int(head.Extension) + 1)
	default:
	}

	out := make([][2]float64, samples)

	for set := range mp3test.Layer1Sets {
		for ch := range channels {
			var subbands, pcm [synth.Subbands]float64

			for sb := range mp3test.Subbands {
				coded := ch
				if sb >= bound {
					coded = 0
				}

				allocation := fields.Allocation[coded][sb]
				if allocation == 0 {
					continue
				}

				width := float64(allocation + 1)
				code := float64(fields.Samples[set][coded][sb])
				factor := math.Pow(2, 1-float64(fields.Scalefactors[ch][sb])/3)

				subbands[sb] = factor * 2 * (code + 1 - math.Pow(2, width-1)) / (math.Pow(2, width) - 1)
			}

			banks[ch].Synthesize(&pcm, &subbands)

			for sample, value := range pcm {
				out[set*synth.Subbands+sample][ch] = value
			}
		}
	}

	if channels == 1 {
		for at := range out {
			out[at][1] = out[at][0]
		}
	}

	return out
}
