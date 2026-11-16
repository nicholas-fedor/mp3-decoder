// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestTags checks that tags before, between and after the frames change no sample.
func TestTags(t *testing.T) {
	t.Parallel()

	data, starts := audioFrames(t, "mpeg1-joint.mp3")
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
// The fixture's tenth frame reaches back to main data in the frames before it, so after junk
// before it, it gives silence and the samples after it change. From the fourth frame after it on,
// they match again.
func TestJunk(t *testing.T) {
	t.Parallel()

	data, starts := audioFrames(t, "mpeg1-joint.mp3")
	want := decodeBytes(t, data)

	junk := make([]byte, 777)
	for at := range junk {
		junk[at] = byte(rand.New(rand.NewPCG(1, uint64(at))).Uint32())
	}

	assert.Equal(t, want, decodeBytes(t, slices.Concat(junk, data)), "junk before the first frame")

	got := decodeBytes(t, insert(data, starts[9], junk))
	before, after := 9*frame.MaxSamples, 13*frame.MaxSamples

	require.Len(t, got, len(want))
	assert.Equal(t, want[:before], got[:before], "before the junk")
	assert.NotEqual(t, want[before:after], got[before:after], "the frames after the junk decode as without it")
	assert.Equal(t, want[after:], got[after:], "from the fourth frame after the tenth")
}

// TestDamagedHeader checks that a frame whose header is damaged is lost alone, and that decoding
// carries on with the frames after it.
//
// The frames after the lost one reach back to its main data, so their samples change. From the
// fourth frame after it on, they match again.
func TestDamagedHeader(t *testing.T) {
	t.Parallel()

	data, starts := audioFrames(t, "mpeg1-joint.mp3")
	want := decodeBytes(t, data)

	damaged := slices.Clone(data)
	clear(damaged[starts[9] : starts[9]+frame.HeaderSize])

	got := decodeBytes(t, damaged)
	before, after := 9*frame.MaxSamples, 13*frame.MaxSamples

	require.Len(t, got, len(want)-frame.MaxSamples)
	assert.Equal(t, want[:before], got[:before], "before the damage")
	assert.Equal(t, want[after:], got[after-frame.MaxSamples:], "from the fourth frame after the tenth")
}

// TestAnotherStream checks that the stream ends where the frames of another stream start, here
// those of an MPEG-2 stream after an MPEG-1 one.
func TestAnotherStream(t *testing.T) {
	t.Parallel()

	first, _ := audioFrames(t, "mpeg1-joint.mp3")
	second, _ := audioFrames(t, "mpeg2.mp3")

	assert.Equal(t, decodeBytes(t, first), decodeBytes(t, slices.Concat(first, second)))
}

// TestJoin opens each fixture partway through and checks that the samples soon match the end of
// the whole decode exactly, as on joining a radio stream.
//
// A first frame needs the header after it to confirm it, and a free-format one needs two. Cuts in
// the last frames leave too few, so they are left out. The free-format fixture is cut more
// sparsely, since each cut measures its frames anew.
func TestJoin(t *testing.T) {
	t.Parallel()

	steps := map[string]int{
		"mpeg1-joint.mp3": 7, "mpeg1-mono.mp3": 7, "protected.mp3": 7, "mpeg2.mp3": 7, "mpeg25.mp3": 3, "free.mp3": 211,
	}

	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, starts := audioFrames(t, name)
			want := decodeBytes(t, data)

			last := len(starts) - 2
			if head, _ := frame.Parse(data[starts[0]:]); head.Free {
				last--
			}

			for cut := 1; cut < starts[last]; cut += step {
				checkJoin(t, data, starts, want, cut)
			}
		})
	}
}

// checkJoin decodes a fixture from a cut and checks it against the whole decode.
//
// Decoding starts at the first frame after the cut and gives every frame's samples from there.
// The frames whose main data began before the cut give silence. Each frame reaches back at most
// 511 bytes in MPEG-1 and 255 at the lower sampling frequencies, so at most that over the smallest
// main data of the fixture's frames are silent. The two filterbanks then take up to two granules,
// at most two frames, to hold only what came after the silence. From there the samples match.
func checkJoin(tb testing.TB, data []byte, starts []int, want [][2]float64, cut int) {
	tb.Helper()

	head, ok := frame.Parse(data[starts[len(starts)-1]:])
	require.True(tb, ok)

	reach, smallest := 511, len(data)
	if head.Version != frame.MPEG1 {
		reach = 255
	}

	for at, start := range starts[:len(starts)-1] {
		this, _ := frame.Parse(data[start:])
		smallest = min(smallest, starts[at+1]-start-this.DataAt())
	}

	got := decodeBytes(tb, data[cut:])
	first, _ := slices.BinarySearch(starts, cut)
	settled := min(((reach+smallest-1)/smallest+2)*head.Samples(), len(got))

	require.Len(tb, got, (len(starts)-first)*head.Samples(), "the samples after a cut at byte %d", cut)
	assert.True(tb, slices.Equal(want[len(want)-len(got)+settled:], got[settled:]),
		"the samples differ from the whole decode %d samples after a cut at byte %d", settled, cut)

	silence := make([][2]float64, head.Samples())

	for at := range silentFrames(tb, data, starts[first:]) {
		assert.True(tb, slices.Equal(silence, got[at*head.Samples():(at+1)*head.Samples()]),
			"frame %d after a cut at byte %d sounds, though its main data began before the cut", at, cut)
	}
}

// silentFrames returns how many frames from the first decoded one begin their main data before it.
//
// Each of them reaches back further than the main data of the frames decoded before it.
func silentFrames(tb testing.TB, data []byte, starts []int) int {
	tb.Helper()

	var side layer3.SideInfo

	held := 0

	for at, start := range starts {
		head, ok := frame.Parse(data[start:])
		require.True(tb, ok)
		require.NoError(tb, side.Parse(head, data[start:]))

		if side.MainDataBegin <= held || at+1 == len(starts) {
			return at
		}

		held += starts[at+1] - start - head.DataAt()
	}

	return len(starts)
}

// audioFrames reads a fixture without its tag frame and finds where each of its frames starts.
//
// Without the tag frame, decoding leaves out none of the samples the encoder added. The frames of a
// free-format fixture take the bitrate measured at its first frame.
func audioFrames(tb testing.TB, name string) ([]byte, []int) {
	tb.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(tb, err)

	at, first, ok := frame.Find(data, nil)
	require.True(tb, ok)

	if _, tagged := container.ParseInfo(first, data[at:]); tagged {
		at += first.Size()
	}

	data = data[at:]

	var starts []int

	for at = 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(tb, ok)

		if head.Free {
			head.Bitrate = first.Bitrate
		}

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
func decodeBytes(tb testing.TB, data []byte) [][2]float64 {
	tb.Helper()

	decoder, err := mp3.OpenStream(tb.Context(), bytes.NewReader(data))
	require.NoError(tb, err)

	return readAll(tb, decoder, 1000)
}
