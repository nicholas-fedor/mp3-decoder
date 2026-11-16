// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzParseWAV parses any bytes as a WAV file and checks that a file it accepts has at least one
// channel, channels of the same length, and samples from -1 up to but not including 1.
//
// The seed corpus holds a plain PCM file, an extensible file with a chunk before fmt, and a file
// with an odd-sized chunk between fmt and data.
func FuzzParseWAV(f *testing.F) {
	data := []byte{0x00, 0x00, 0x80, 0xff, 0xff, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xff}

	f.Add(riff(fmtChunk(pcm24(2)), chunk("data", data)))
	f.Add(riff(chunk("JUNK", []byte{1}), fmtChunk(extensible24(1)), chunk("data", data)))
	f.Add(riff(fmtChunk(pcm24(1)), chunk("LIST", []byte{1, 2, 3}), chunk("data", data[:9])))

	f.Fuzz(func(t *testing.T, file []byte) {
		wave, err := ParseWAV(file)
		if err != nil {
			require.ErrorIs(t, err, ErrWAV)

			return
		}

		require.NotEmpty(t, wave.Channels)

		for _, channel := range wave.Channels {
			require.Len(t, channel, len(wave.Channels[0]))

			for _, sample := range channel {
				assert.GreaterOrEqual(t, sample, -1.0)
				assert.Less(t, sample, 1.0)
			}
		}
	})
}

// FuzzMP4Frames reads any bytes as an MP4 file and checks that it never panics and fails only
// with ErrMP4.
//
// The seed corpus holds a file of listed sample sizes in runs of chunks, one of uniform sizes with
// 64-bit offsets, and one cut short.
func FuzzMP4Frames(f *testing.F) {
	samples := [][]byte{[]byte("one"), []byte("two"), []byte("three")}

	f.Add(mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 2}, {2, 1}}, chunks: 2}))
	f.Add(mp4File(mp4Layout{samples: samples[:2], runs: [][2]uint32{{1, 1}}, chunks: 2, uniform: true, wide: true}))
	f.Add(mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 3}}, chunks: 1})[:40])

	f.Fuzz(func(t *testing.T, file []byte) {
		_, err := mp4Frames(file)
		if err != nil {
			require.ErrorIs(t, err, ErrMP4)
		}
	})
}
