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

// FuzzDecode decodes a Layer I frame of any bytes, twice in a row, in any mode and with or
// without a CRC.
//
// It checks that the frame decodes or fails with [layer12.ErrAllocation], that every sample is
// finite, and that a mono frame gives each sample to both channels. The seed corpus holds a frame
// of random fields in each mode, a frame with the forbidden allocation, and an empty frame.
func FuzzDecode(f *testing.F) {
	random := rand.New(rand.NewPCG(1, 2))

	for mode := range uint8(4) {
		fields := mp3test.RandomLayer1Audio(random, 14)
		head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: uint32(mode), Extension: uint32(mode)}

		f.Add(head.Layer1Audio(&fields), mode, mode, mode%2 == 0)
	}

	f.Add([]byte{0xf0}, uint8(mp3test.Mono), uint8(0), false)
	f.Add([]byte{}, uint8(mp3test.Stereo), uint8(0), true)

	f.Fuzz(func(t *testing.T, audio []byte, mode, extension uint8, protected bool) {
		head := mp3test.Header{
			Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: 14, Mode: uint32(mode % 4),
			Extension: uint32(extension % 4), Protected: protected,
		}

		parsed, ok := frame.Parse(head.Bytes())
		require.True(t, ok)

		data := append(head.Frame(parsed.DataAt()), audio...)
		pcm := make([][2]float64, frame.MaxSamples)

		var decoder layer12.Decoder

		for range 2 {
			_, err := decoder.Decode(pcm, parsed, data)
			if err != nil {
				require.ErrorIs(t, err, layer12.ErrAllocation)

				continue
			}

			for at, pair := range pcm[:mp3test.Layer1Sets*mp3test.Subbands] {
				require.False(t, math.IsNaN(pair[0]) || math.IsInf(pair[0], 0), "sample %d", at)
				require.False(t, math.IsNaN(pair[1]) || math.IsInf(pair[1], 0), "sample %d", at)

				if parsed.Channels() == 1 {
					require.InDelta(t, pair[0], pair[1], 0, "sample %d", at)
				}
			}
		}
	})
}
