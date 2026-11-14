// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTagBytes checks encoded tags field by field.
//
// The delays are the LAME tag specification's own example, 1729 and 722 samples.
func TestTagBytes(t *testing.T) {
	t.Parallel()

	toc := make([]byte, 100)
	toc[1] = 9

	tests := []struct {
		name string
		tag  Tag
		want []byte
	}{
		{
			"frame count and size",
			Tag{Magic: "Xing", Flags: HasFrames | HasBytes, Frames: 7, Size: 4000},
			[]byte{'X', 'i', 'n', 'g', 0, 0, 0, 3, 0, 0, 0, 7, 0, 0, 0x0f, 0xa0},
		},
		{
			"seek table and quality",
			Tag{Magic: "Info", Flags: HasTOC | HasQuality, TOC: toc, Quality: 50},
			slices.Concat([]byte{'I', 'n', 'f', 'o', 0, 0, 0, 0x0c}, toc, []byte{0, 0, 0, 50}),
		},
		{
			"LAME extension",
			Tag{Magic: "Info", Encoder: "LAME3.100", Delay: 1729, Padding: 722},
			slices.Concat(
				[]byte{'I', 'n', 'f', 'o', 0, 0, 0, 0},
				[]byte("LAME3.100"), make([]byte, 12), []byte{0x6c, 0x12, 0xd2},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.tag.Bytes())
		})
	}
}

// TestVBRIBytes checks an encoded VBRI tag field by field.
func TestVBRIBytes(t *testing.T) {
	t.Parallel()

	want := []byte{'V', 'B', 'R', 'I', 0, 1, 0, 0, 0, 0, 0, 0, 0x13, 0x88, 0, 0, 0, 9}

	assert.Equal(t, want, VBRI{Version: 1, Size: 5000, Frames: 9}.Bytes())
}
