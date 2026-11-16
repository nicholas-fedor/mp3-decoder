// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// ID3v2 holds the shape of an ID3v2 tag.
type ID3v2 struct {
	// Body is the size of the tag's frames.
	Body int
	// Footer adds a footer.
	Footer bool
}

// APE holds the shape of an APEv2 tag.
type APE struct {
	// Items is the size of the tag's items.
	Items int
	// Header adds a header before the items.
	Header bool
}

// ID3v2 tags.
const (
	// id3Version is the major version written.
	id3Version = 4
	// id3FooterFlag marks a tag with a footer.
	id3FooterFlag = 0x10
	// syncsafeBits is the number of bits each byte of a syncsafe integer holds.
	syncsafeBits = 7
	// syncsafeMask keeps the bits of one byte of a syncsafe integer.
	syncsafeMask = 0x7f
	// syncsafeBytes is the number of bytes in a syncsafe size.
	syncsafeBytes = 4
)

// ID3v1, APEv2 and Lyrics3 v2.00 tags.
const (
	// id3v1Size is the size of an ID3v1 tag.
	id3v1Size = 128
	// apeVersion is the APE tag version written.
	apeVersion = 2000
	// apeFooterSize is the size of an APE footer and header.
	apeFooterSize = 32
	// apeHasHeader flags a tag with a header.
	apeHasHeader = 1 << 31
	// apeIsHeader flags a header rather than a footer.
	apeIsHeader = 1 << 29
	// lyricsBegin starts a Lyrics3 tag.
	lyricsBegin = "LYRICSBEGIN"
	// lyricsEnd ends a Lyrics3 v2.00 tag.
	lyricsEnd = "LYRICS200"
)

// Bytes encodes the tag with a body of zeros.
//
// Returns:
//   - []byte: the header, the body and the footer when there is one.
func (t ID3v2) Bytes() []byte {
	var flags byte

	if t.Footer {
		flags = id3FooterFlag
	}

	header := append([]byte("ID3"), id3Version, 0, flags)
	for shift := (syncsafeBytes - 1) * syncsafeBits; shift >= 0; shift -= syncsafeBits {
		header = append(header, byte(t.Body>>shift&syncsafeMask))
	}

	out := slices.Concat(header, make([]byte, t.Body))
	if t.Footer {
		out = append(out, "3DI"...)
		out = append(out, header[len("ID3"):]...)
	}

	return out
}

// Bytes encodes the tag with items of zeros.
//
// Returns:
//   - []byte: the header when there is one, the items and the footer.
func (a APE) Bytes() []byte {
	var flags uint32

	if a.Header {
		flags = apeHasHeader
	}

	var out []byte

	if a.Header {
		out = apeBlock(a.Items, flags|apeIsHeader)
	}

	out = append(out, make([]byte, a.Items)...)

	return append(out, apeBlock(a.Items, flags)...)
}

// ID3v1 builds an ID3v1 tag of zeros.
//
// Returns:
//   - []byte: the tag.
func ID3v1() []byte {
	return append([]byte("TAG"), make([]byte, id3v1Size-len("TAG"))...)
}

// Lyrics3 builds a Lyrics3 v2.00 tag whose fields hold body bytes of zeros.
//
// Returns:
//   - []byte: the tag.
func Lyrics3(body int) []byte {
	tag := append([]byte(lyricsBegin), make([]byte, body)...)

	return fmt.Appendf(tag, "%06d%s", len(tag), lyricsEnd)
}

// apeBlock builds an APE header or footer.
//
// Parameters:
//   - items: the size of the tag's items.
//   - flags: the block's flags.
//
// Returns:
//   - []byte: the block.
func apeBlock(items int, flags uint32) []byte {
	out := binary.LittleEndian.AppendUint32([]byte("APETAGEX"), apeVersion)

	out = binary.LittleEndian.AppendUint32(out, uint32(items+apeFooterSize)) //nolint:gosec // Test tags are small.
	out = binary.LittleEndian.AppendUint32(out, 0)
	out = binary.LittleEndian.AppendUint32(out, flags)

	return append(out, make([]byte, apeFooterSize-len(out))...)
}
