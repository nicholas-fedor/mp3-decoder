// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// layer1Head is a protected MPEG-1 Layer I header in joint stereo from subband 8, at 44.1 kHz and
// 448 kbit/s.
var layer1Head = mp3test.Header{
	Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: 14, Mode: mp3test.JointStereo, Extension: 1, Protected: true,
}

// TestLayer1 checks that a Layer I stream opens, gives its length and decodes every frame.
func TestLayer1(t *testing.T) {
	t.Parallel()

	data := layer1Stream(t, 20)

	stream, err := mp3.OpenStream(t.Context(), bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, mp3.Info{SampleRate: 44100, Channels: 2}, stream.Info())

	samples := readAll(t, stream, 1000)
	assert.Len(t, samples, 20*384)
	assert.Equal(t, mp3.Stats{Frames: 20}, stream.Stats())
	assert.NotEqual(t, make([][2]float64, len(samples)), samples)

	file := openFile(t, data)
	length, known := file.Len()
	assert.True(t, known)
	assert.Equal(t, int64(20*384), length)
}

// layer1Stream builds a stream of Layer I frames of random fields, each with its CRC.
//
// Each allocation is at most 3, so the samples of 4 bits at most fit the frame.
func layer1Stream(tb testing.TB, count int) []byte {
	tb.Helper()

	head, ok := frame.Parse(layer1Head.Bytes())
	require.True(tb, ok)

	random := rand.New(rand.NewPCG(11, 12))
	out := make([]byte, 0, count*head.Size())

	for range count {
		fields := mp3test.RandomLayer1Audio(random, 3)
		data := mp3test.Place(layer1Head.Frame(head.Size()), head.DataAt(), layer1Head.Layer1Audio(&fields))

		// The CRC covers 16 allocations below subband 8 and 24 from it on, of 4 bits each.
		out = append(out, mp3test.Protect(data, 4*(16+24))...)
	}

	return out
}
