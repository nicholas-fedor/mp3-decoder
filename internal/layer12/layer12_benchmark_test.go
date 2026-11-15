// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer12"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// BenchmarkDecode measures decoding stereo Layer I frames whose every subband holds samples.
func BenchmarkDecode(b *testing.B) {
	random := rand.New(rand.NewPCG(1, 2))
	frames := make([][]byte, 8)

	var head frame.Header

	for at := range frames {
		fields := mp3test.RandomLayer1Audio(random, 14)

		for ch := range fields.Allocation {
			for sb := range fields.Allocation[ch] {
				fields.Allocation[ch][sb] = max(fields.Allocation[ch][sb], 1)
			}
		}

		frames[at], head = layer1Frame(b, mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.Stereo}, &fields)
	}

	var decoder layer12.Decoder

	pcm := make([][2]float64, frame.MaxSamples)

	b.ReportAllocs()

	for b.Loop() {
		for _, data := range frames {
			_, err := decoder.Decode(pcm, head, data)
			require.NoError(b, err)
		}
	}
}
