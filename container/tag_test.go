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

// TestTagSize covers each tag a stream may carry between its frames, with audio after it, and
// bytes that start with no such tag.
func TestTagSize(t *testing.T) {
	t.Parallel()

	audio := []byte{0xff, 0xfb, 0x90, 0x00, 1, 2, 3}

	tests := []struct {
		name string
		head []byte
		want int
	}{
		{"ID3v2", slices.Concat(mp3test.ID3v2{Body: 300}.Bytes(), audio), 300 + ID3v2HeaderSize},
		{"ID3v2 with a footer", slices.Concat(mp3test.ID3v2{Body: 300, Footer: true}.Bytes(), audio), 300 + 2*ID3v2HeaderSize},
		{"ID3v1", slices.Concat(mp3test.ID3v1(), audio), id3v1Size},
		{"APE with a header", slices.Concat(mp3test.APE{Items: 40, Header: true}.Bytes(), audio), 40 + 2*apeFooterSize},
		{"APE without a header", slices.Concat(mp3test.APE{Items: 40}.Bytes(), audio), 0},
		{"a frame", audio, 0},
		{"short", []byte("TA"), 0},
		{"none", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TagSize(tt.head))
		})
	}
}

// TestAPEHeaderRejects covers APE headers whose fields rule them out, a footer, which starts no
// tag, and a header cut short.
func TestAPEHeaderRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		damage func(header []byte)
	}{
		{"newer version", func(h []byte) { binary.LittleEndian.PutUint32(h[apeVersionAt:], apeMaxVersion+1) }},
		{"size below the footer", func(h []byte) { binary.LittleEndian.PutUint32(h[apeSizeAt:], apeFooterSize-1) }},
		{"size too large", func(h []byte) { binary.LittleEndian.PutUint32(h[apeSizeAt:], apeMaxSize+1) }},
		{"a footer, not a header", func(h []byte) { binary.LittleEndian.PutUint32(h[apeFlagsAt:], apeHasHeader) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag := mp3test.APE{Items: 8, Header: true}.Bytes()
			tt.damage(tag[:apeFooterSize])

			assert.Zero(t, apeHeader(tag))
		})
	}

	assert.Zero(t, apeHeader(mp3test.APE{Items: 8, Header: true}.Bytes()[:apeFooterSize-1]), "short")
}
