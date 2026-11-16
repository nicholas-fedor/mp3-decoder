// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"encoding/binary"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestTrailerSize covers each tag alone, the tags in the orders taggers write them, and tails that
// end in no tag or in a broken one.
func TestTrailerSize(t *testing.T) {
	t.Parallel()

	audio := []byte{0xff, 0xfb, 0x90, 0x00, 1, 2, 3}

	tests := []struct {
		name string
		tail []byte
		want int
	}{
		{"none", audio, 0},
		{"ID3v1", slices.Concat(audio, mp3test.ID3v1()), id3v1Size},
		{"APE with a header", slices.Concat(audio, mp3test.APE{Items: 40, Header: true}.Bytes()), 40 + 2*apeFooterSize},
		{"APE without a header", slices.Concat(audio, mp3test.APE{Items: 40}.Bytes()), 40 + apeFooterSize},
		{"Lyrics3", slices.Concat(audio, mp3test.Lyrics3(20)), 20 + len(lyricsBegin) + lyricsDigits + len(lyricsEnd)},
		{
			"APE then ID3v1", slices.Concat(audio, mp3test.APE{Items: 40, Header: true}.Bytes(), mp3test.ID3v1()),
			40 + 2*apeFooterSize + id3v1Size,
		},
		{
			"Lyrics3, APE, then ID3v1", slices.Concat(audio, mp3test.Lyrics3(20), mp3test.APE{Items: 8}.Bytes(), mp3test.ID3v1()),
			20 + len(lyricsBegin) + lyricsDigits + len(lyricsEnd) + 8 + apeFooterSize + id3v1Size,
		},
		{"APE larger than the tail", mp3test.APE{Items: 40, Header: true}.Bytes()[apeFooterSize+20:], 40 + 2*apeFooterSize},
		{"Lyrics3 without its begin marker", slices.Concat(audio, mp3test.Lyrics3(20)[1:]), 0},
		{"Lyrics3 larger than the tail", mp3test.Lyrics3(20)[5:], 0},
		{"Lyrics3 with a size that is not a number", slices.Concat(audio, []byte("LYRICSBEGIN00x0aaLYRICS200")), 0},
		{"short", []byte("TAG"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TrailerSize(tt.tail))
		})
	}
}

// TestAPETagRejects covers APE footers whose fields rule them out.
func TestAPETagRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		damage func(footer []byte)
	}{
		{"newer version", func(f []byte) { binary.LittleEndian.PutUint32(f[apeVersionAt:], apeMaxVersion+1) }},
		{"size below the footer", func(f []byte) { binary.LittleEndian.PutUint32(f[apeSizeAt:], apeFooterSize-1) }},
		{"size too large", func(f []byte) { binary.LittleEndian.PutUint32(f[apeSizeAt:], apeMaxSize+1) }},
		{"a header, not a footer", func(f []byte) { binary.LittleEndian.PutUint32(f[apeFlagsAt:], apeIsHeader) }},
		{"no magic", func(f []byte) { f[0] = 'X' }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag := mp3test.APE{Items: 8}.Bytes()
			tt.damage(tag[len(tag)-apeFooterSize:])

			assert.Zero(t, apeTag(tag))
		})
	}
}
