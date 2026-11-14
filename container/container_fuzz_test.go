// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// smallestTrailer is the size of the smallest tag TrailerSize takes, an empty Lyrics3 v2.00 tag.
const smallestTrailer = 26

// FuzzParseInfo checks that a parsed tag's fields lie in range, that its seek table is a copy and
// that its offsets lie within the stream.
//
// The seed corpus holds the first frame of each fixture and a VBRI frame.
func FuzzParseInfo(f *testing.F) {
	for _, file := range []string{"cbr.mp3", "vbr.mp3", "protected.mp3", "mpeg2.mp3"} {
		data, err := os.ReadFile("testdata/" + file)
		require.NoError(f, err)

		f.Add(data[:min(len(data), 600)])
	}

	f.Add(tagged(plainHead, mp3test.VBRIAt, mp3test.VBRI{Version: vbriVersion, Size: 5000, Frames: 9}.Bytes()))

	f.Fuzz(func(t *testing.T, data []byte) {
		head, ok := frame.Parse(data)
		if !ok {
			return
		}

		info, ok := ParseInfo(head, data)
		if !ok {
			return
		}

		assert.GreaterOrEqual(t, info.Frames, int64(0))
		assert.GreaterOrEqual(t, info.Bytes, int64(0))
		assert.True(t, info.TOC == nil || len(info.TOC) == tocSize)
		assert.True(t, info.Gapless || info.Delay == 0 && info.Padding == 0)
		assert.LessOrEqual(t, info.Delay, 1<<12-1)
		assert.LessOrEqual(t, info.Padding, 1<<12-1)
		assert.GreaterOrEqual(t, info.Samples(head), int64(0))

		if info.TOC == nil {
			return
		}

		toc := slices.Clone(info.TOC)
		clear(data)
		assert.Equal(t, toc, info.TOC)

		for _, fraction := range []float64{0, 0.33, 0.5, 1} {
			offset, ok := info.OffsetAt(fraction, 1<<20)
			require.True(t, ok)
			assert.GreaterOrEqual(t, offset, int64(0))
			assert.LessOrEqual(t, offset, int64(1<<20))
		}
	})
}

// FuzzTrailerSize checks that a trailer size is 0 or at least the smallest tag.
//
// The seed corpus holds each tag alone and the tags in the order taggers write them.
func FuzzTrailerSize(f *testing.F) {
	for _, tail := range [][]byte{
		mp3test.ID3v1(),
		mp3test.APE{Items: 40, Header: true}.Bytes(),
		mp3test.Lyrics3(20),
		slices.Concat(mp3test.Lyrics3(4), mp3test.APE{Items: 8}.Bytes(), mp3test.ID3v1()),
	} {
		f.Add(tail)
	}

	f.Fuzz(func(t *testing.T, tail []byte) {
		size := TrailerSize(tail)

		assert.True(t, size == 0 || size >= smallestTrailer, "size %d", size)
	})
}

// FuzzID3v2Size checks that a tag size is 0, or at least a header's size for bytes that start with
// the magic.
//
// The seed corpus holds a tag with a footer and one without.
func FuzzID3v2Size(f *testing.F) {
	f.Add(mp3test.ID3v2{Body: 257}.Bytes())
	f.Add(mp3test.ID3v2{Body: 257, Footer: true}.Bytes())

	f.Fuzz(func(t *testing.T, head []byte) {
		size := ID3v2Size(head)
		if size == 0 {
			return
		}

		assert.GreaterOrEqual(t, size, id3HeaderSize)
		assert.Equal(t, id3Magic, string(head[:len(id3Magic)]))
	})
}
