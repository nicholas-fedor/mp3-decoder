// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
)

// TestStats checks the counts of a clean stream, of a damaged CRC, of junk between frames and of
// a stream joined midway. It also checks those of a clean Layer II stream with CRCs and a damaged
// Layer II CRC, then a clean Layer I stream, a damaged Layer I CRC and a Layer I frame with the
// forbidden allocation.
//
// The underflows after junk or a join are the frames whose main data begins before it, as their
// side information gives. Every frame of the fixture that LAME encoded with CRCs passes its CRC. A
// frame whose CRC is damaged decodes all the same, so the samples do not change. The CRC of a
// Layer I frame covers its allocations, so the forbidden allocation fails the CRC too, and the
// frame gives silence.
func TestStats(t *testing.T) {
	t.Parallel()

	protected, starts := audioFrames(t, "protected.mp3")
	frames := int64(len(starts))

	damaged := slices.Clone(protected)
	damaged[starts[5]+frame.HeaderSize] ^= 0xff

	layer2, layer2Starts := audioFrames(t, "layer2-joint.mp2")

	layer2Damaged := slices.Clone(layer2)
	layer2Damaged[layer2Starts[5]+frame.HeaderSize] ^= 0xff

	layer1 := layer1Stream(t, 10)
	layer1Size := len(layer1) / 10

	layer1Damaged := slices.Clone(layer1)
	layer1Damaged[5*layer1Size+frame.HeaderSize] ^= 0xff

	// An allocation of 15 in the first subband of the left channel, which no encoder writes.
	layer1Forbidden := slices.Clone(layer1)
	layer1Forbidden[3*layer1Size+frame.HeaderSize+2] |= 0xf0

	joint, jointStarts := audioFrames(t, "mpeg1-joint.mp3")
	junk := slices.Concat(joint[:jointStarts[6]], []byte("junk!"), joint[jointStarts[6]:])

	tests := []struct {
		name string
		data []byte
		want mp3.Stats
	}{
		{"clean", protected, mp3.Stats{Frames: frames}},
		{"a damaged CRC", damaged, mp3.Stats{Frames: frames, CRCMismatches: 1}},
		{"junk", junk, mp3.Stats{
			Frames: int64(len(jointStarts)), Resyncs: 1, Underflows: reachingBefore(t, joint, jointStarts, 6),
		}},
		{"joined midway", joint[jointStarts[3]:], mp3.Stats{
			Frames: int64(len(jointStarts) - 3), Underflows: reachingBefore(t, joint, jointStarts, 3),
		}},
		{"a clean Layer II stream with CRCs", layer2, mp3.Stats{Frames: int64(len(layer2Starts))}},
		{"a damaged Layer II CRC", layer2Damaged, mp3.Stats{Frames: int64(len(layer2Starts)), CRCMismatches: 1}},
		{"a clean Layer I stream", layer1, mp3.Stats{Frames: 10}},
		{"a damaged Layer I CRC", layer1Damaged, mp3.Stats{Frames: 10, CRCMismatches: 1}},
		{"a forbidden Layer I allocation", layer1Forbidden, mp3.Stats{Frames: 10, CRCMismatches: 1, Undecodable: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := mp3.OpenStream(t.Context(), bytes.NewReader(tt.data))
			require.NoError(t, err)

			samples := readAll(t, decoder, 1000)
			assert.Equal(t, tt.want, decoder.Stats())

			switch tt.name {
			case "a damaged CRC":
				assert.True(t, slices.Equal(decodeBytes(t, protected), samples), "the samples change")
			case "a damaged Layer II CRC":
				assert.True(t, slices.Equal(decodeBytes(t, layer2), samples), "the samples change")
			case "a damaged Layer I CRC":
				assert.True(t, slices.Equal(decodeBytes(t, layer1), samples), "the samples change")
			case "a forbidden Layer I allocation":
				assert.Equal(t, make([][2]float64, 384), samples[3*384:4*384], "the frame is not silent")
			default:
			}
		})
	}
}

// TestStatsSeek checks that the frames a seek decodes before its sample are not counted, though
// the first of them may give silence, while the frame that holds the sample is.
func TestStatsSeek(t *testing.T) {
	t.Parallel()

	data, starts := audioFrames(t, "mpeg1-joint.mp3")
	decoder := openFile(t, data)

	require.NoError(t, decoder.Seek(t.Context(), 20*1152+5))
	assert.Equal(t, mp3.Stats{Frames: 1}, decoder.Stats())

	readAll(t, decoder, 1000)
	assert.Equal(t, mp3.Stats{Frames: int64(len(starts) - 20)}, decoder.Stats())
}

// reachingBefore counts the frames from one on whose main data begins before the main data of
// that one, which the decoder lacks after junk or a join there.
//
// The bit reservoir holds up to 511 bytes of the main data from that frame on.
func reachingBefore(tb testing.TB, data []byte, starts []int, from int) int64 {
	tb.Helper()

	var (
		side  layer3.SideInfo
		count int64
	)

	held := 0

	for _, at := range starts[from:] {
		head, ok := frame.Parse(data[at:])
		require.True(tb, ok)
		require.NoError(tb, side.Parse(head, data[at:]))

		if side.MainDataBegin > held {
			count++
		}

		held = min(held+head.Size()-head.DataAt(), 511)
	}

	return count
}
