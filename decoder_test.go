// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// Headers of the streams the tests build.
var (
	// stereoHead is an MPEG-1 Layer III stereo header at 128 kbit/s.
	stereoHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	// lsfHead is an MPEG-2 Layer III header at 64 kbit/s.
	lsfHead = mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8}
	// layer2Head is an MPEG-1 Layer II header at 112 kbit/s.
	layer2Head = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer2, Bitrate: 8}
	// freeHead is an MPEG-1 Layer III free-format header.
	freeHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3}
)

// errSource is the error the failing sources return.
var errSource = errors.New("source failed")

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
		{bytes.NewReader(frames(layer2Head, 3)), ErrLayer, "Layer II", false},
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

// TestReadEnds checks where a stream ends, and that a frame that fails to decode is reported and
// moved past.
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

		got, err := readSamples(decoder)
		require.ErrorIs(t, err, layer3.ErrBlockType)
		assert.Len(t, got, 2*frame.MaxSamples)

		got, err = readSamples(decoder)
		require.ErrorIs(t, err, io.EOF)
		assert.Len(t, got, 2*frame.MaxSamples)
	})

	t.Run("frames of another version", func(t *testing.T) {
		t.Parallel()

		decoder, err := OpenStream(t.Context(), bytes.NewReader(frames(lsfHead, 3)))
		require.NoError(t, err)

		_, err = readSamples(decoder)
		require.ErrorIs(t, err, layer3.ErrVersion)
	})

	for name, size := range map[string]int{"a source failing within a frame": 1000, "a source failing between frames": 834} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			decoder, err := OpenStream(t.Context(), failAfter(frames(stereoHead, 3)[:size]))
			require.NoError(t, err)

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, errSource)
			assert.Len(t, got, 2*frame.MaxSamples)
		})
	}
}

// frames builds a count of silent frames of a header.
//
// A free-format header takes frames of 417 bytes.
func frames(head mp3test.Header, count int) []byte {
	size := 417

	if parsed, _ := frame.Parse(head.Bytes()); !parsed.Free {
		size = parsed.Size()
	}

	out := make([]byte, 0, size*count)

	for range count {
		out = append(out, head.Frame(size)...)
	}

	return out
}

// failAfter returns a source that hands out data, then fails.
func failAfter(data []byte) io.Reader {
	return io.MultiReader(bytes.NewReader(data), iotest.ErrReader(errSource))
}

// readSamples reads a decoder until an error, returning the samples before it.
func readSamples(decoder *Decoder) ([][2]float64, error) {
	var out [][2]float64

	buffer := make([][2]float64, 1000)

	for {
		read, err := decoder.Read(buffer)
		out = append(out, buffer[:read]...)

		if err != nil {
			return out, err
		}
	}
}
