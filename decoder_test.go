// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3

import (
	"bytes"
	"context"
	"io"
	"slices"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestOpenStream checks streams that open, with what comes before their first frame, and the
// number of samples each holds.
func TestOpenStream(t *testing.T) {
	t.Parallel()

	tag := frames(stereoHead, 1)
	mp3test.Place(tag, frame.HeaderSize+32, mp3test.Tag{Magic: "Info", Flags: mp3test.HasFrames, Frames: 3}.Bytes())

	tests := []struct {
		name    string
		data    []byte
		samples int
	}{
		{"frames alone", frames(stereoHead, 3), 3 * frame.MaxSamples},
		{"an ID3v2 tag larger than a read", slices.Concat(mp3test.ID3v2{Body: 3 * readSize}.Bytes(), frames(stereoHead, 3)), 3 * frame.MaxSamples},
		{"junk before the first frame", slices.Concat(bytes.Repeat([]byte{0xff, 0x00}, 3000), frames(stereoHead, 3)), 3 * frame.MaxSamples},
		{"an Info frame", slices.Concat(tag, frames(stereoHead, 3)), 3 * frame.MaxSamples},
		{"free format", frames(freeHead, 4), 4 * frame.MaxSamples},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := OpenStream(t.Context(), bytes.NewReader(tt.data))
			require.NoError(t, err)

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, io.EOF)
			assert.Len(t, got, tt.samples)
			assert.Equal(t, make([][2]float64, tt.samples), got, "silence")
		})
	}
}

// TestOpenStreamErrors checks the streams that do not open.
func TestOpenStreamErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		source   io.Reader
		want     error
		name     string
		canceled bool
	}{
		{bytes.NewReader(bytes.Repeat([]byte{0xff}, 5000)), ErrNoFrames, "no frames", false},
		{bytes.NewReader(frames(stereoHead, 3)), context.Canceled, "canceled", true},
		{iotest.ErrReader(errSource), errSource, "failing source", false},
		{bytes.NewReader(mp3test.ID3v2{Body: 3 * readSize}.Bytes()[:100]), ErrNoFrames, "a tag longer than the stream", false},
		{failAfter(mp3test.ID3v2{Body: 3 * readSize}.Bytes()[:100]), errSource, "failing in a tag", false},
		{failAfter(bytes.Repeat([]byte{0xff}, 100)), errSource, "failing in a search", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tt.canceled {
				cancel()
			}

			decoder, err := OpenStream(ctx, tt.source)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, decoder)
		})
	}
}

// TestOpen checks the length a file reports, from its tag frame when the tag's byte count matches
// the audio and from a walk over its frames otherwise, and the samples it decodes to.
//
// A tag whose byte count matches is trusted even when its frame count is wrong, as the first
// case of a matching tag shows.
func TestOpen(t *testing.T) {
	t.Parallel()

	// The tag follows the header and the 32 bytes of stereo side information.
	tagged := func(tag mp3test.Tag) []byte {
		first := frames(stereoHead, 1)
		mp3test.Place(first, frame.HeaderSize+32, tag.Bytes())

		return first
	}
	counts := uint32(mp3test.HasFrames | mp3test.HasBytes)

	tests := []struct {
		name    string
		data    []byte
		length  int64
		samples int
	}{
		{"no tag frame", frames(stereoHead, 3), 3 * frame.MaxSamples, 3 * frame.MaxSamples},
		{
			"a tag frame whose byte count matches", slices.Concat(tagged(mp3test.Tag{Magic: "Info", Flags: counts, Frames: 5, Size: 4 * 417}), frames(stereoHead, 3)),
			5 * frame.MaxSamples, 3 * frame.MaxSamples,
		},
		{
			"a tag frame whose byte count differs", slices.Concat(tagged(mp3test.Tag{Magic: "Info", Flags: counts, Frames: 5, Size: 999}), frames(stereoHead, 3)),
			3 * frame.MaxSamples, 3 * frame.MaxSamples,
		},
		{
			"a tag frame with no frame count", slices.Concat(tagged(mp3test.Tag{Magic: "Info", Flags: mp3test.HasBytes, Size: 4 * 417}), frames(stereoHead, 3)),
			3 * frame.MaxSamples, 3 * frame.MaxSamples,
		},
		{
			"encoder padding", slices.Concat(tagged(mp3test.Tag{Magic: "Info", Flags: counts, Frames: 3, Size: 4 * 417, Encoder: "LAME3.100", Delay: 576, Padding: 1000}), frames(stereoHead, 3)),
			3*frame.MaxSamples - 576 - 1000, 3*frame.MaxSamples - 576 - 1000,
		},
		{
			"tags after the audio", slices.Concat(frames(stereoHead, 3), mp3test.APE{Items: 40, Header: true}.Bytes(), mp3test.ID3v1()),
			3 * frame.MaxSamples, 3 * frame.MaxSamples,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := Open(t.Context(), bytes.NewReader(tt.data))
			require.NoError(t, err)

			length, known := decoder.Len()
			assert.True(t, known)
			assert.Equal(t, tt.length, length, "length")

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, io.EOF)
			assert.Len(t, got, tt.samples, "samples")
		})
	}

	decoder, err := OpenStream(t.Context(), bytes.NewReader(frames(stereoHead, 3)))
	require.NoError(t, err)

	_, known := decoder.Len()
	assert.False(t, known, "a stream knows its length")
}

// TestInfo checks the sampling frequency and channels of streams at each version.
func TestInfo(t *testing.T) {
	t.Parallel()

	mono := lsfHead
	mono.Mode = mp3test.Mono

	tests := []struct {
		name string
		data []byte
		want Info
	}{
		{"MPEG-1 stereo", frames(stereoHead, 3), Info{SampleRate: 44100, Channels: 2}},
		{"MPEG-2 mono", frames(mono, 3), Info{SampleRate: 22050, Channels: 1}},
	}

	for _, tt := range tests {
		decoder, err := OpenStream(t.Context(), bytes.NewReader(tt.data))
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.want, decoder.Info(), tt.name)
	}
}

// TestOpenErrors checks the files that do not open: seeks and reads that fail, files without
// frames or of another layer, and a canceled search.
func TestOpenErrors(t *testing.T) {
	t.Parallel()

	long := frames(stereoHead, 30)

	tests := []struct {
		file     io.ReadSeeker
		want     error
		name     string
		canceled bool
	}{
		{&failingSeeker{Reader: bytes.NewReader(long), seeks: 0}, errSource, "the seek to the end", false},
		{&failingSeeker{Reader: bytes.NewReader(long), seeks: 1}, errSource, "the seek to the tail", false},
		{&failingSeeker{Reader: bytes.NewReader(long), seeks: 2}, errSource, "the seek to the start", false},
		{&failingSeeker{Reader: bytes.NewReader(long), seeks: 3}, errSource, "the seek back after the walk", false},
		{&failingReader{Reader: bytes.NewReader(long), reads: 100}, errSource, "the read of the tail", false},
		{&failingReader{Reader: bytes.NewReader(long), reads: len(long) + readSize + 100}, errSource, "a read in the walk", false},
		{bytes.NewReader(bytes.Repeat([]byte{0xff}, 5000)), ErrNoFrames, "no frames", false},
		{bytes.NewReader(long), context.Canceled, "canceled", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tt.canceled {
				cancel()
			}

			decoder, err := Open(ctx, tt.file)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, decoder)
		})
	}
}

// TestSeekErrors checks that a seek stops at a failing seek or read of the file, or at a canceled
// walk.
//
// The stream's tag frame gives its length, so Open walks none of its frames and a seek far into
// it walks them. Open reads the last 64 KiB and then one read of the start.
func TestSeekErrors(t *testing.T) {
	t.Parallel()

	first := frames(stereoHead, 1)
	mp3test.Place(first, frame.HeaderSize+32, mp3test.Tag{
		Magic: "Info", Flags: uint32(mp3test.HasFrames | mp3test.HasBytes), Frames: 200, Size: 201 * 417,
	}.Bytes())

	trusted := slices.Concat(first, frames(stereoHead, 200))

	tests := []struct {
		file     io.ReadSeeker
		name     string
		canceled bool
	}{
		{&failingSeeker{Reader: bytes.NewReader(frames(stereoHead, 200)), seeks: 4}, "a seek of the file", false},
		{&failingReader{Reader: bytes.NewReader(trusted), reads: tailSize + 3*readSize}, "a read in the walk", false},
		// The walk ends at frame 141, 9 frames before the sample's, which ends at byte 59631.
		{&failingReader{Reader: bytes.NewReader(trusted), reads: tailSize + 60000}, "a read in the decoding", false},
		{bytes.NewReader(trusted), "canceled", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := Open(t.Context(), tt.file)
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tt.canceled {
				cancel()
			}

			err = decoder.Seek(ctx, 150*frame.MaxSamples)
			if tt.canceled {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, errSource)
			}
		})
	}
}

// TestReadEnds checks where a stream ends, and that a frame that fails to decode gives silence in
// its place.
func TestReadEnds(t *testing.T) {
	t.Parallel()

	broken := frames(stereoHead, 1)
	mp3test.Place(broken, frame.HeaderSize, stereoHead.SideInfo(&mp3test.SideInfo{
		Granules: [2][2]mp3test.Granule{{{WindowSwitching: true, BlockType: 0}}},
	}))

	tests := []struct {
		name    string
		data    []byte
		samples int
	}{
		{"an ID3v1 trailer", slices.Concat(frames(stereoHead, 3), mp3test.ID3v1()), 3 * frame.MaxSamples},
		{"a frame cut short", slices.Concat(frames(stereoHead, 3), frames(stereoHead, 1)[:200]), 4 * frame.MaxSamples},
		{"a frame cut in its side information", slices.Concat(frames(stereoHead, 3), frames(stereoHead, 1)[:35]), 3 * frame.MaxSamples},
		{"another stream", slices.Concat(frames(stereoHead, 3), frames(lsfHead, 3)), 3 * frame.MaxSamples},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := OpenStream(t.Context(), bytes.NewReader(tt.data))
			require.NoError(t, err)

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, io.EOF)
			assert.Len(t, got, tt.samples)
		})
	}

	t.Run("a frame that fails to decode", func(t *testing.T) {
		t.Parallel()

		decoder, err := OpenStream(t.Context(), bytes.NewReader(slices.Concat(frames(stereoHead, 2), broken, frames(stereoHead, 2))))
		require.NoError(t, err)

		read, err := decoder.Read(make([][2]float64, 2*frame.MaxSamples))
		require.NoError(t, err)
		require.Equal(t, 2*frame.MaxSamples, read)

		// The broken frame must not hand out the samples of the frame before it.
		for at := range decoder.pcm {
			decoder.pcm[at] = [2]float64{1, 1}
		}

		got, err := readSamples(decoder)
		require.ErrorIs(t, err, io.EOF)
		require.Len(t, got, 3*frame.MaxSamples)
		assert.Equal(t, make([][2]float64, frame.MaxSamples), got[:frame.MaxSamples])
		assert.Equal(t, Stats{Frames: 5, Resyncs: 0, CRCMismatches: 0, Underflows: 0, Undecodable: 1}, decoder.Stats())
	})

	t.Run("MPEG-2 frames", func(t *testing.T) {
		t.Parallel()

		decoder, err := OpenStream(t.Context(), bytes.NewReader(frames(lsfHead, 3)))
		require.NoError(t, err)

		got, err := readSamples(decoder)
		require.ErrorIs(t, err, io.EOF)
		assert.Len(t, got, 3*frame.MaxSamples/2)
	})

	t.Run("MPEG-2 intensity stereo", func(t *testing.T) {
		t.Parallel()

		intensity := lsfHead
		intensity.Mode, intensity.Extension = mp3test.JointStereo, 1

		decoder, err := OpenStream(t.Context(), bytes.NewReader(frames(intensity, 3)))
		require.NoError(t, err)

		got, err := readSamples(decoder)
		require.ErrorIs(t, err, io.EOF)
		assert.Len(t, got, 3*frame.MaxSamples/2)
	})

	sources := map[string][]byte{
		"a source failing within a frame":   frames(stereoHead, 3)[:1000],
		"a source failing between frames":   frames(stereoHead, 3)[:834],
		"a source failing after junk bytes": slices.Concat(frames(stereoHead, 2), []byte("junk!")),
	}

	for name, data := range sources {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			decoder, err := OpenStream(t.Context(), failAfter(data))
			require.NoError(t, err)

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, errSource)
			assert.Len(t, got, 2*frame.MaxSamples)
		})
	}
}
