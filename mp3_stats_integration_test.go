// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

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
// a stream joined midway.
//
// The underflows after junk or a join are the frames whose main data begins before it, as their
// side information gives. Every frame of the fixture that LAME encoded with CRCs passes its CRC. A
// frame whose CRC is damaged decodes all the same, so the samples do not change.
func TestStats(t *testing.T) {
	t.Parallel()

	protected, starts := audioFrames(t, "protected.mp3")
	frames := int64(len(starts))

	damaged := slices.Clone(protected)
	damaged[starts[5]+frame.HeaderSize] ^= 0xff

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := mp3.OpenStream(t.Context(), bytes.NewReader(tt.data))
			require.NoError(t, err)

			samples := readAll(t, decoder, 1000)
			assert.Equal(t, tt.want, decoder.Stats())

			if tt.name == "a damaged CRC" {
				assert.True(t, slices.Equal(decodeBytes(t, protected), samples), "the samples change")
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
