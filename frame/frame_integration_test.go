// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestEncodedFile walks the frames of a CRC-protected file that LAME encoded.
//
// Find confirms the first frame. Each frame after it parses as the same stream, and each CRC
// matches.
func TestEncodedFile(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/protected.mp3")
	require.NoError(t, err)

	at, first, confirmed := frame.Find(data, nil)
	require.True(t, confirmed)
	require.Zero(t, at)

	frames := 0

	for at < len(data) {
		head, ok := frame.Parse(data[at:])
		require.True(t, ok, "frame %d at %d", frames, at)
		require.True(t, first.SameStream(head))

		end := at + head.Size()
		require.LessOrEqual(t, end, len(data))
		assert.True(t, head.CheckCRC(data[at:end], head.SideInfoSize()*8), "frame %d", frames)

		at = end
		frames++
	}

	assert.Greater(t, frames, 2)
}

// TestFreeFormatStream walks a free-format stream after junk, sizing each frame by the bitrate
// Find measured and the frame's own padding.
func TestFreeFormatStream(t *testing.T) {
	t.Parallel()

	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Protected: true}
	padded := head
	padded.Padding = true

	sizes := []int{500, 501, 500, 500, 501}
	stream := []byte{0x00, 0xff, 0x12}

	for _, size := range sizes {
		if size == 501 {
			stream = append(stream, padded.Frame(size)...)
		} else {
			stream = append(stream, head.Frame(size)...)
		}
	}

	at, first, confirmed := frame.Find(stream, nil)
	require.True(t, confirmed)
	require.Equal(t, 3, at)
	require.True(t, first.Free)

	var walked []int

	for at < len(stream) {
		next, ok := frame.Parse(stream[at:])
		require.True(t, ok)

		next.Bitrate = first.Bitrate
		walked = append(walked, next.Size())
		at += next.Size()
	}

	assert.Equal(t, sizes, walked)
	assert.Equal(t, len(stream), at)
}
