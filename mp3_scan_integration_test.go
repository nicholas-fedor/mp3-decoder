// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"math/rand/v2"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestTags checks that tags before, between and after the frames change no sample.
func TestTags(t *testing.T) {
	t.Parallel()

	data, starts := frameStarts(t, "mpeg1-joint.mp3")
	want := decodeBytes(t, data)
	tags := slices.Concat(
		mp3test.ID3v2{Body: 300, Footer: true}.Bytes(), mp3test.ID3v1(), mp3test.APE{Items: 40, Header: true}.Bytes(),
	)

	tests := []struct {
		name   string
		stream []byte
	}{
		{"before", slices.Concat(mp3test.ID3v2{Body: 100}.Bytes(), mp3test.ID3v2{Body: 50}.Bytes(), tags, data)},
		{"between frames", insert(insert(data, starts[9], tags), starts[5], tags)},
		{"after", slices.Concat(data, mp3test.APE{Items: 40}.Bytes(), mp3test.Lyrics3(20), mp3test.ID3v1())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, want, decodeBytes(t, tt.stream))
		})
	}
}

// TestJunk checks that junk before the first frame changes no sample, and that junk between two
// frames loses none.
//
// Frame 10 of the fixture reaches back to main data in the frames before it, so after junk before
// it, it gives silence and the samples after it change. From the fourth frame after it on, they
// match again.
func TestJunk(t *testing.T) {
	t.Parallel()

	data, starts := frameStarts(t, "mpeg1-joint.mp3")
	want := decodeBytes(t, data)

	junk := make([]byte, 777)
	for at := range junk {
		junk[at] = byte(rand.New(rand.NewPCG(1, uint64(at))).Uint32())
	}

	assert.Equal(t, want, decodeBytes(t, slices.Concat(junk, data)), "junk before the first frame")

	got := decodeBytes(t, insert(data, starts[10], junk))

	// The first frame holds the Info tag, so frame 10 gives samples from the ninth frame's on.
	before, after := 9*frame.MaxSamples, 13*frame.MaxSamples

	require.Len(t, got, len(want))
	assert.Equal(t, want[:before], got[:before], "before the junk")
	assert.NotEqual(t, want[before:after], got[before:after], "the frames after the junk decode as without it")
	assert.Equal(t, want[after:], got[after:], "from the fourth frame after frame 10")
}

// TestDamagedHeader checks that a frame whose header is damaged is lost alone, and that decoding
// carries on with the frames after it.
//
// The frames after the lost one reach back to its main data, so their samples change. From the
// fourth frame after it on, they match again.
func TestDamagedHeader(t *testing.T) {
	t.Parallel()

	data, starts := frameStarts(t, "mpeg1-joint.mp3")
	want := decodeBytes(t, data)

	damaged := slices.Clone(data)
	clear(damaged[starts[10] : starts[10]+frame.HeaderSize])

	got := decodeBytes(t, damaged)
	before, after := 9*frame.MaxSamples, 13*frame.MaxSamples

	require.Len(t, got, len(want)-frame.MaxSamples)
	assert.Equal(t, want[:before], got[:before], "before the damage")
	assert.Equal(t, want[after:], got[after-frame.MaxSamples:], "from the fourth frame after frame 10")
}

// TestAnotherStream checks that the stream ends where the frames of another stream start, here
// those of an MPEG-2 stream after an MPEG-1 one.
func TestAnotherStream(t *testing.T) {
	t.Parallel()

	first, _ := frameStarts(t, "mpeg1-joint.mp3")
	second, _ := frameStarts(t, "mpeg2.mp3")

	assert.Equal(t, decodeBytes(t, first), decodeBytes(t, slices.Concat(first, second)))
}

// frameStarts reads a fixture and finds where each of its frames starts, its Info frame first.
func frameStarts(t *testing.T, name string) ([]byte, []int) {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	at, _, ok := frame.Find(data, nil)
	require.True(t, ok)

	var starts []int

	for at < len(data) {
		head, ok := frame.Parse(data[at:])
		require.True(t, ok)

		starts = append(starts, at)
		at += head.Size()
	}

	return data, starts
}

// insert returns a copy of data with extra bytes at an offset.
func insert(data []byte, at int, extra []byte) []byte {
	return slices.Concat(data[:at], extra, data[at:])
}

// decodeBytes decodes a stream held in memory with reads of 1000 samples.
func decodeBytes(t *testing.T, data []byte) [][2]float64 {
	t.Helper()

	decoder, err := mp3.OpenStream(t.Context(), bytes.NewReader(data))
	require.NoError(t, err)

	return readAll(t, decoder, 1000)
}
