// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// TestDecodeLayer2 checks frames of every allocation table and mode against the requantization
// formula and the synthesis filterbank, sample by sample.
//
// Each test decodes three frames in turn, so the filterbanks carry each frame into the next. The
// table of 8 subbands in joint stereo from subband 16 puts the bound above the subbands it codes.
func TestDecodeLayer2(t *testing.T) {
	t.Parallel()

	mpeg1 := func(rate, bitrate, mode, extension uint32) mp3test.Header {
		return mp3test.Header{
			Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: bitrate, Rate: rate, Mode: mode, Extension: extension,
		}
	}

	tests := []struct {
		name string
		head mp3test.Header
	}{
		{"27 subbands in stereo", mpeg1(0, 8, mp3test.Stereo, 0)},
		{"27 subbands in joint stereo from subband 8", mpeg1(0, 8, mp3test.JointStereo, 1)},
		{"30 subbands in dual channel", mpeg1(0, 10, mp3test.DualChannel, 0)},
		{"30 subbands in mono", mpeg1(0, 6, mp3test.Mono, 0)},
		{"8 subbands in joint stereo from subband 16", mpeg1(0, 4, mp3test.JointStereo, 3)},
		{"12 subbands at 32 kHz in joint stereo from subband 4", mpeg1(2, 4, mp3test.JointStereo, 0)},
		{"lower rates in joint stereo from subband 12", mp3test.Header{
			Version: mp3test.MPEG2, Layer: mp3test.Layer2, Bitrate: 8, Mode: mp3test.JointStereo, Extension: 2,
		}},
		{"lower rates in mono with a CRC", mp3test.Header{
			Version: mp3test.MPEG2, Layer: mp3test.Layer2, Bitrate: 8, Mode: mp3test.Mono, Protected: true,
		}},
		{"48 kHz with a CRC", mp3test.Header{
			Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: 12, Rate: 1, Mode: mp3test.Stereo, Protected: true,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				decoder Decoder
				banks   [2]synth.Filterbank
			)

			random := rand.New(rand.NewPCG(1, 2))
			got := make([][2]float64, frame.MaxSamples)

			for at := range 3 {
				data, head, fields := layer2Frame(t, tt.head, random)

				intact, err := decoder.Decode(got, head, data)
				require.NoError(t, err)
				assert.True(t, intact, "CRC")

				want := layer2Reference(head, &fields, &banks)

				for sample := range frame.MaxSamples {
					require.InDelta(t, want[sample][0], got[sample][0], 1e-12, "frame %d sample %d", at, sample)
					require.InDelta(t, want[sample][1], got[sample][1], 1e-12, "frame %d sample %d", at, sample)
				}
			}
		})
	}
}

// TestDecodeLayer2CRC checks that a frame whose CRC does not match its bit allocation and
// selections decodes all the same, to the samples of the frame with its CRC intact.
func TestDecodeLayer2CRC(t *testing.T) {
	t.Parallel()

	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: 8, Mode: mp3test.JointStereo, Protected: true}
	data, parsed, _ := layer2Frame(t, head, rand.New(rand.NewPCG(3, 4)))

	broken := slices.Clone(data)
	broken[frame.HeaderSize] ^= 1

	var intact, damaged Decoder

	want := make([][2]float64, frame.MaxSamples)
	got := make([][2]float64, frame.MaxSamples)

	matched, err := intact.Decode(want, parsed, data)
	require.NoError(t, err)
	require.True(t, matched)

	matched, err = damaged.Decode(got, parsed, broken)
	require.NoError(t, err)
	assert.False(t, matched)
	assert.Equal(t, want, got)
}

// TestReadCodes checks codes written alone and grouped, and that a grouped word that holds more
// than three samples' codes gives none.
func TestReadCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		class quantClass
		value uint32
		width int
		want  [granuleSets]int
		valid bool
	}{
		{"alone", steps7, 0b110_000_101, 9, [granuleSets]int{6, 0, 5}, true},
		{"grouped", steps3, 2 + 3*1 + 9*2, 5, [granuleSets]int{2, 1, 2}, true},
		{"the largest grouped word", steps9, 728, 10, [granuleSets]int{8, 8, 8}, true},
		{"a grouped word past three codes", steps3, 27, 5, [granuleSets]int{}, false},
		{"the largest word of 5 steps", steps5, 127, 7, [granuleSets]int{}, false},
	}

	for _, tt := range tests {
		var writer mp3test.BitWriter

		writer.Write(tt.value, tt.width)

		decoder := Decoder{stream: bits.Reader{}}
		decoder.stream.Reset(writer.Bytes(), 0)

		codes, valid := decoder.readCodes(tt.class)
		assert.Equal(t, tt.want, codes, tt.name)
		assert.Equal(t, tt.valid, valid, tt.name)
		assert.Equal(t, tt.width, decoder.stream.Pos(), tt.name)
	}
}

// TestReadGranuleUngroupable checks that the samples of a word past three codes are 0.
func TestReadGranuleUngroupable(t *testing.T) {
	t.Parallel()

	var writer mp3test.BitWriter

	writer.Write(31, steps3.bits)

	var decoder Decoder

	decoder.classes[0][0] = steps3
	decoder.steps[0][0] = 1
	decoder.samples[1][0][0] = 0.5
	decoder.stream.Reset(writer.Bytes(), 0)
	decoder.readGranule(1, 1, synth.Subbands)

	assert.Equal(t, [granuleSets]float64{}, [granuleSets]float64{
		decoder.samples[0][0][0], decoder.samples[1][0][0], decoder.samples[2][0][0],
	})
}

// TestReadLayer2Scalefactors checks the scalefactor each part takes under each selection.
func TestReadLayer2Scalefactors(t *testing.T) {
	t.Parallel()

	want := map[uint32][layer2Parts]uint32{
		eachPart: {1, 2, 3}, firstPair: {1, 1, 2}, allParts: {1, 1, 1}, lastPair: {1, 2, 2},
	}

	for selection, factors := range want {
		var writer mp3test.BitWriter

		writer.Write(1, scalefactorBits)
		writer.Write(2, scalefactorBits)
		writer.Write(3, scalefactorBits)

		var decoder Decoder

		decoder.classes[0][0], decoder.selections[0][0] = steps3, selection
		decoder.stream.Reset(writer.Bytes(), 0)
		decoder.readLayer2Scalefactors(1, 1)

		assert.Equal(t, factors, decoder.factors[0][0], "selection %d", selection)
	}
}

// layer2Frame builds a frame of a Layer II header holding random audio data that follows the
// frame's allocation table, with its CRC when the header is protected.
//
// The frame holds the audio data whole, even where it is larger than a frame of the bitrate.
func layer2Frame(tb testing.TB, head mp3test.Header, random *rand.Rand) ([]byte, frame.Header, mp3test.Layer2Audio) {
	tb.Helper()

	parsed, ok := frame.Parse(head.Bytes())
	require.True(tb, ok)

	var table []mp3test.Layer2Subband

	for _, run := range allocationFor(parsed) {
		steps := make([]int, 0, len(run.classes))
		for _, class := range run.classes {
			steps = append(steps, class.steps)
		}

		for range run.subbands {
			table = append(table, mp3test.Layer2Subband{Steps: steps, Width: run.width})
		}
	}

	fields := mp3test.RandomLayer2Audio(random, table)
	audio := head.Layer2Audio(&fields)
	data := mp3test.Place(head.Frame(max(parsed.Size(), parsed.DataAt()+len(audio))), parsed.DataAt(), audio)

	if head.Protected {
		mp3test.Protect(data, protectedBits(parsed, &fields))
	}

	return data, parsed, fields
}

// protectedBits returns the number of bits of a Layer II frame's audio data that its CRC covers:
// each allocation, and the selection of each subband and channel with samples.
func protectedBits(head frame.Header, fields *mp3test.Layer2Audio) int {
	covered := 0

	for sb, subband := range fields.Table {
		for ch := range head.Channels() {
			coded := 0
			if sb < layer2Bound(head) {
				coded = ch
			}

			if coded == ch {
				covered += subband.Width
			}

			if fields.Allocation[coded][sb] != 0 {
				covered += 2
			}
		}
	}

	return covered
}

// layer2Bound returns the first subband whose samples both channels share, from the mode
// extension of a joint stereo frame.
func layer2Bound(head frame.Header) int {
	if head.Mode != frame.JointStereo {
		return synth.Subbands
	}

	return 4 * (int(head.Extension) + 1)
}

// layer2Reference decodes a Layer II frame by the requantization formula and the synthesis
// filterbank.
//
// A sample with code c of a class of N steps and scalefactor index i in its part is 2^(1-i/3)
// times (2c-(N-1))/N. From the bound on, both channels take the first channel's allocation and
// codes, each with its own scalefactors.
func layer2Reference(head frame.Header, fields *mp3test.Layer2Audio, banks *[2]synth.Filterbank) [][2]float64 {
	out := make([][2]float64, frame.MaxSamples)

	for set := range mp3test.Layer2Sets {
		for ch := range head.Channels() {
			var subbands, pcm [synth.Subbands]float64

			for sb, subband := range fields.Table {
				coded := 0
				if sb < layer2Bound(head) {
					coded = ch
				}

				allocation := fields.Allocation[coded][sb]
				if allocation == 0 {
					continue
				}

				steps := float64(subband.Steps[allocation-1])
				code := float64(fields.Samples[set][coded][sb])
				factor := math.Pow(2, 1-float64(fields.Scalefactors[ch][sb][set/12])/3)

				subbands[sb] = factor * (2*code - (steps - 1)) / steps
			}

			banks[ch].Synthesize(&pcm, &subbands)

			for sample, value := range pcm {
				out[set*synth.Subbands+sample][ch] = value
			}
		}
	}

	if head.Mode == frame.Mono {
		for at := range out {
			out[at][1] = out[at][0]
		}
	}

	return out
}
