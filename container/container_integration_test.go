// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container_test

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestEncodedFiles checks the tags LAME writes for a source of exactly 10000 samples, which
// Samples must give back.
func TestEncodedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file   string
		frames int64
	}{
		{"cbr.mp3", 10},
		{"vbr.mp3", 10},
		{"protected.mp3", 10},
		{"mpeg2.mp3", 20},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile("testdata/" + tt.file)
			require.NoError(t, err)

			head, ok := frame.Parse(data)
			require.True(t, ok)

			info, ok := container.ParseInfo(head, data)
			require.True(t, ok)

			assert.Equal(t, tt.frames, info.Frames)
			assert.Equal(t, int64(len(data)), info.Bytes)
			assert.Len(t, info.TOC, 100)
			assert.True(t, info.Gapless)
			assert.Equal(t, 576, info.Delay)
			assert.Equal(t, 944, info.Padding)
			assert.Equal(t, int64(10000), info.Samples(head))
		})
	}
}

// TestTaggedFile finds the audio of a file wrapped in an ID3v2 tag and three trailing tags, then
// reads the tag frame at its start.
func TestTaggedFile(t *testing.T) {
	t.Parallel()

	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	tag := mp3test.Tag{
		Magic: "Info", Flags: mp3test.HasFrames | mp3test.HasBytes, Frames: 3, Size: 4 * 417,
		Encoder: "LAME3.100", Delay: 576, Padding: 100,
	}

	audio := slices.Concat(mp3test.Place(head.Frame(417), 36, tag.Bytes()), head.Frame(417), head.Frame(417), head.Frame(417))
	file := slices.Concat(
		mp3test.ID3v2{Body: 300, Footer: true}.Bytes(), audio,
		mp3test.APE{Items: 50, Header: true}.Bytes(), mp3test.Lyrics3(10), mp3test.ID3v1(),
	)

	start := container.ID3v2Size(file)
	end := len(file) - container.TrailerSize(file)
	assert.Equal(t, audio, file[start:end])

	at, first, confirmed := frame.Find(file[start:end], nil)
	require.True(t, confirmed)
	require.Zero(t, at)

	info, ok := container.ParseInfo(first, file[start:])
	require.True(t, ok)

	assert.Equal(t, int64(end-start), info.Bytes)
	assert.Equal(t, int64(3*1152-676), info.Samples(first))
}
