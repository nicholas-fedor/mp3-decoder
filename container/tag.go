// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package container

import "encoding/binary"

// TagHeadSize is the most bytes [TagSize] reads to recognize a tag.
const TagHeadSize = apeFooterSize

// TagSize returns the size of the tag at the start of head.
//
// It recognizes an ID3v2 tag, an ID3v1 tag, and an APEv2 tag that starts with its header. A
// stream may carry any of them between its frames, as a tag before each track of a joined stream
// or at its end.
//
// Parameters:
//   - head: the bytes, of which the first [TagHeadSize] are read.
//
// Returns:
//   - int: the size of the whole tag, or 0 when head does not start with one of these tags.
func TagSize(head []byte) int {
	size := ID3v2Size(head)

	switch {
	case size > 0:
		return size
	case len(head) >= len(id3v1Magic) && string(head[:len(id3v1Magic)]) == id3v1Magic:
		return id3v1Size
	default:
		return apeHeader(head)
	}
}

// apeHeader returns the size of the APEv2 tag whose header starts head.
//
// Parameters:
//   - head: the bytes.
//
// Returns:
//   - int: the size in bytes with the header and footer, or 0 when head starts with no header.
func apeHeader(head []byte) int {
	if len(head) < apeFooterSize || string(head[:len(apeMagic)]) != apeMagic {
		return 0
	}

	version := binary.LittleEndian.Uint32(head[apeVersionAt:])
	size := binary.LittleEndian.Uint32(head[apeSizeAt:])
	flags := binary.LittleEndian.Uint32(head[apeFlagsAt:])

	if version > apeMaxVersion || size < apeFooterSize || size > apeMaxSize || flags&apeIsHeader == 0 {
		return 0
	}

	return int(size) + apeFooterSize
}
