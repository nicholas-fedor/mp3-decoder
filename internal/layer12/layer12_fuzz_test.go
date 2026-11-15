// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer12"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// FuzzDecode decodes a Layer I or Layer II frame of any bytes, twice in a row, in any mode, at any
// bitrate of either version and with or without a CRC.
//
// It checks that a Layer I frame decodes or fails with [layer12.ErrAllocation] and a Layer II frame
// decodes, that every sample is finite, and that a mono frame gives each sample to both channels.
// The seed corpus holds a Layer I frame of random fields in each mode, a frame with the forbidden
// allocation, an empty frame, and Layer II frames of random bytes at several bitrates.
func FuzzDecode(f *testing.F) {
	random := rand.New(rand.NewPCG(1, 2))

	for mode := range uint8(4) {
		fields := mp3test.RandomLayer1Audio(random, 14)
		head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: uint32(mode), Extension: uint32(mode)}

		f.Add(head.Layer1Audio(&fields), mode, mode, uint8(14), mode%2 == 0, false, false)
	}

	f.Add([]byte{0xf0}, uint8(mp3test.Mono), uint8(0), uint8(14), false, false, false)
	f.Add([]byte{}, uint8(mp3test.Stereo), uint8(0), uint8(14), true, false, false)

	for _, bitrate := range []uint8{4, 8, 10, 14} {
		audio := make([]byte, 600)
		for at := range audio {
			audio[at] = byte(random.Uint32())
		}

		f.Add(audio, bitrate%4, bitrate%4, bitrate, bitrate == 8, true, bitrate == 4)
	}

	f.Fuzz(func(t *testing.T, audio []byte, mode, extension, bitrate uint8, protected, layer2, lower bool) {
		head := mp3test.Header{
			Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: uint32(bitrate%14 + 1), Mode: uint32(mode % 4),
			Extension: uint32(extension % 4), Protected: protected,
		}

		if layer2 {
			head.Layer = mp3test.Layer2
		}

		if lower {
			head.Version = mp3test.MPEG2
		}

		parsed, ok := frame.Parse(head.Bytes())
		require.True(t, ok)

		data := append(head.Frame(parsed.DataAt()), audio...)
		pcm := make([][2]float64, frame.MaxSamples)

		var decoder layer12.Decoder

		for range 2 {
			_, err := decoder.Decode(pcm, parsed, data)
			if err != nil {
				require.False(t, layer2)
				require.ErrorIs(t, err, layer12.ErrAllocation)

				continue
			}

			for at, pair := range pcm[:parsed.Samples()] {
				require.False(t, math.IsNaN(pair[0]) || math.IsInf(pair[0], 0), "sample %d", at)
				require.False(t, math.IsNaN(pair[1]) || math.IsInf(pair[1], 0), "sample %d", at)

				if parsed.Channels() == 1 {
					require.InDelta(t, pair[0], pair[1], 0, "sample %d", at)
				}
			}
		}
	})
}
