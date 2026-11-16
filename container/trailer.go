// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"encoding/binary"
	"strconv"
)

// ID3v1 tags.
const (
	// id3v1Magic starts an ID3v1 tag.
	id3v1Magic = "TAG"
	// id3v1Size is the size of an ID3v1 tag.
	id3v1Size = 128
)

// APEv2 tags.
const (
	// apeMagic starts an APE tag's footer and header.
	apeMagic = "APETAGEX"
	// apeFooterSize is the size of an APE tag's footer and of its optional header.
	apeFooterSize = 32
	// apeVersionAt is the offset of the version in the footer.
	apeVersionAt = 8
	// apeSizeAt is the offset of the tag size in the footer, which counts the items and the footer.
	apeSizeAt = 12
	// apeFlagsAt is the offset of the flags in the footer.
	apeFlagsAt = 20
	// apeMaxVersion is the highest version a tag may have.
	apeMaxVersion = 2000
	// apeMaxSize is the largest tag size taken.
	apeMaxSize = 16<<20 + apeFooterSize
	// apeHasHeader flags a tag that has a header.
	apeHasHeader = 1 << 31
	// apeIsHeader flags a header rather than a footer.
	apeIsHeader = 1 << 29
)

// Lyrics3 v2.00 tags.
const (
	// lyricsBegin starts a tag.
	lyricsBegin = "LYRICSBEGIN"
	// lyricsEnd ends a tag.
	lyricsEnd = "LYRICS200"
	// lyricsDigits is the number of decimal digits in the size before the end marker.
	lyricsDigits = 6
)

// TrailerSize returns the size of the tags at the end of a stream.
//
// It removes ID3v1, APEv2 and Lyrics3 v2.00 tags in any order. An APE tag whose footer gives a size
// beyond tail ends the search.
//
// Parameters:
//   - tail: the last bytes of the stream.
//
// Returns:
//   - int: the size of the tags in bytes, or 0 when the stream ends with none.
func TrailerSize(tail []byte) int {
	total := 0

	for total < len(tail) {
		rest := tail[:len(tail)-total]

		size := id3v1Tag(rest)
		if size == 0 {
			size = apeTag(rest)
		}

		if size == 0 {
			size = lyricsTag(rest)
		}

		if size == 0 {
			break
		}

		total += size
	}

	return total
}

// id3v1Tag returns the size of the ID3v1 tag that rest ends with.
//
// Parameters:
//   - rest: the bytes.
//
// Returns:
//   - int: the size in bytes, or 0 when rest ends with no tag.
func id3v1Tag(rest []byte) int {
	if len(rest) < id3v1Size || string(rest[len(rest)-id3v1Size:][:len(id3v1Magic)]) != id3v1Magic {
		return 0
	}

	return id3v1Size
}

// apeTag returns the size of the APEv2 tag whose footer rest ends with.
//
// Parameters:
//   - rest: the bytes.
//
// Returns:
//   - int: the size in bytes with the header, which may exceed rest, or 0 when rest ends with no
//     footer.
func apeTag(rest []byte) int {
	if len(rest) < apeFooterSize {
		return 0
	}

	footer := rest[len(rest)-apeFooterSize:]
	if string(footer[:len(apeMagic)]) != apeMagic {
		return 0
	}

	version := binary.LittleEndian.Uint32(footer[apeVersionAt:])
	size := binary.LittleEndian.Uint32(footer[apeSizeAt:])
	flags := binary.LittleEndian.Uint32(footer[apeFlagsAt:])

	if version > apeMaxVersion || size < apeFooterSize || size > apeMaxSize || flags&apeIsHeader != 0 {
		return 0
	}

	if flags&apeHasHeader != 0 {
		size += apeFooterSize
	}

	return int(size)
}

// lyricsTag returns the size of the Lyrics3 v2.00 tag that rest ends with.
//
// Parameters:
//   - rest: the bytes.
//
// Returns:
//   - int: the size in bytes, or 0 when rest ends with no whole tag.
func lyricsTag(rest []byte) int {
	trailer := lyricsDigits + len(lyricsEnd)
	if len(rest) < trailer || string(rest[len(rest)-len(lyricsEnd):]) != lyricsEnd {
		return 0
	}

	body, err := strconv.Atoi(string(rest[len(rest)-trailer : len(rest)-len(lyricsEnd)]))
	if err != nil || body < len(lyricsBegin) || body+trailer > len(rest) {
		return 0
	}

	start := len(rest) - trailer - body
	if string(rest[start:start+len(lyricsBegin)]) != lyricsBegin {
		return 0
	}

	return body + trailer
}
