// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container

// ID3v2 tag headers.
const (
	// id3Magic starts an ID3v2 tag.
	id3Magic = "ID3"
	// id3HeaderSize is the size of a tag's header and of its optional footer.
	id3HeaderSize = 10
	// id3VersionAt is the offset of the major version.
	id3VersionAt = 3
	// id3RevisionAt is the offset of the revision.
	id3RevisionAt = 4
	// id3FlagsAt is the offset of the flags.
	id3FlagsAt = 5
	// id3SizeAt is the offset of the four-byte syncsafe size.
	id3SizeAt = 6
	// id3FooterFlag marks a tag with a footer.
	id3FooterFlag = 0x10
	// id3Invalid is the version or revision byte no tag has.
	id3Invalid = 0xff
	// syncsafeBits is the number of bits each byte of a syncsafe integer holds.
	syncsafeBits = 7
	// syncsafeHigh is the bit no byte of a syncsafe integer sets.
	syncsafeHigh = 0x80
)

// ID3v2Size returns the size of the ID3v2 tag at the start of head.
//
// Parameters:
//   - head: the bytes, of which the first 10 are read.
//
// Returns:
//   - int: the size of the whole tag with its header and footer, or 0 when head does not start
//     with a tag.
func ID3v2Size(head []byte) int {
	if len(head) < id3HeaderSize || string(head[:len(id3Magic)]) != id3Magic {
		return 0
	}

	if head[id3VersionAt] == id3Invalid || head[id3RevisionAt] == id3Invalid {
		return 0
	}

	size := 0

	for _, digit := range head[id3SizeAt:id3HeaderSize] {
		if digit&syncsafeHigh != 0 {
			return 0
		}

		size = size<<syncsafeBits | int(digit)
	}

	size += id3HeaderSize
	if head[id3FlagsAt]&id3FooterFlag != 0 {
		size += id3HeaderSize
	}

	return size
}
