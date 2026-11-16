// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestID3v2Bytes checks the syncsafe size and the footer of encoded tags.
func TestID3v2Bytes(t *testing.T) {
	t.Parallel()

	tag := ID3v2{Body: 257}.Bytes()

	assert.Len(t, tag, 10+257)
	assert.Equal(t, []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0x02, 0x01}, tag[:10])

	footed := ID3v2{Body: 257, Footer: true}.Bytes()

	assert.Len(t, footed, 20+257)
	assert.Equal(t, byte(0x10), footed[5])
	assert.Equal(t, []byte{'3', 'D', 'I', 4, 0, 0x10, 0, 0, 0x02, 0x01}, footed[len(footed)-10:])
}

// TestAPEBytes checks the sizes and flags of encoded tags with and without a header.
func TestAPEBytes(t *testing.T) {
	t.Parallel()

	tag := APE{Items: 40}.Bytes()
	footer := tag[40:]

	assert.Len(t, tag, 40+32)
	assert.Equal(t, "APETAGEX", string(footer[:8]))
	assert.Equal(t, uint32(2000), binary.LittleEndian.Uint32(footer[8:]))
	assert.Equal(t, uint32(72), binary.LittleEndian.Uint32(footer[12:]))
	assert.Zero(t, binary.LittleEndian.Uint32(footer[20:]))

	headed := APE{Items: 40, Header: true}.Bytes()

	assert.Len(t, headed, 40+64)
	assert.Equal(t, uint32(1<<31|1<<29), binary.LittleEndian.Uint32(headed[20:]))
	assert.Equal(t, uint32(1<<31), binary.LittleEndian.Uint32(headed[len(headed)-12:]))
}

// TestID3v1 checks the size and magic of an ID3v1 tag.
func TestID3v1(t *testing.T) {
	t.Parallel()

	tag := ID3v1()

	assert.Len(t, tag, 128)
	assert.Equal(t, "TAG", string(tag[:3]))
}

// TestLyrics3 checks that the size before the end marker counts the begin marker and the fields.
func TestLyrics3(t *testing.T) {
	t.Parallel()

	tag := Lyrics3(20)

	assert.Len(t, tag, 31+15)
	assert.Equal(t, "LYRICSBEGIN", string(tag[:11]))
	assert.Equal(t, "000031LYRICS200", string(tag[31:]))
}
