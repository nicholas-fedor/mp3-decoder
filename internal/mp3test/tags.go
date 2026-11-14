// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import "encoding/binary"

// Tag holds the fields of a Xing or Info tag and its LAME extension.
type Tag struct {
	// Magic is "Xing" or "Info".
	Magic string
	// Encoder names the encoder in the LAME extension, or is empty for a tag without one.
	Encoder string
	// TOC is the seek table, written when Flags names it.
	TOC []byte
	// Flags names the fields the tag holds.
	Flags uint32
	// Frames is the frame count.
	Frames uint32
	// Size is the stream size.
	Size uint32
	// Quality is the quality indicator.
	Quality uint32
	// Delay is the encoder delay in the LAME extension.
	Delay int
	// Padding is the padding in the LAME extension.
	Padding int
}

// VBRI holds the fields of a VBRI tag.
type VBRI struct {
	// Version is the tag's version.
	Version uint16
	// Size is the stream size.
	Size uint32
	// Frames is the frame count.
	Frames uint32
}

// Flags of a Xing or Info tag.
const (
	// HasFrames flags the frame count.
	HasFrames = 0x1
	// HasBytes flags the stream size.
	HasBytes = 0x2
	// HasTOC flags the seek table.
	HasTOC = 0x4
	// HasQuality flags the quality indicator.
	HasQuality = 0x8
	// HasAll flags every field.
	HasAll = HasFrames | HasBytes | HasTOC | HasQuality
)

// The LAME extension and the VBRI tag.
const (
	// lameSize is the size of the extension written.
	lameSize = 24
	// lameDelayAt is the offset of the encoder delay and padding in the extension.
	lameDelayAt = 21
	// paddingBits is the width of the padding, which follows the 12-bit delay.
	paddingBits = 12
	// vbriMagic starts a VBRI tag.
	vbriMagic = "VBRI"
	// vbriSkipped is the size of the delay and quality fields after the version, written as zeros.
	vbriSkipped = 4
)

// VBRIAt is the offset of a VBRI tag from its frame's start.
const VBRIAt = 36

// Bytes encodes the tag.
//
// Returns:
//   - []byte: the magic, the flags, the fields the flags name and the LAME extension.
func (t Tag) Bytes() []byte {
	out := binary.BigEndian.AppendUint32([]byte(t.Magic), t.Flags)

	if t.Flags&HasFrames != 0 {
		out = binary.BigEndian.AppendUint32(out, t.Frames)
	}

	if t.Flags&HasBytes != 0 {
		out = binary.BigEndian.AppendUint32(out, t.Size)
	}

	if t.Flags&HasTOC != 0 {
		out = append(out, t.TOC...)
	}

	if t.Flags&HasQuality != 0 {
		out = binary.BigEndian.AppendUint32(out, t.Quality)
	}

	if t.Encoder == "" {
		return out
	}

	var ext [lameSize]byte

	copy(ext[:], t.Encoder)

	// The delays take three bytes, so the byte before them stays zero.
	delays := uint32(t.Delay)<<paddingBits | uint32(t.Padding) //nolint:gosec // Both are 12-bit values.
	binary.BigEndian.PutUint32(ext[lameDelayAt-1:], delays)

	return append(out, ext[:]...)
}

// Bytes encodes the tag.
//
// Returns:
//   - []byte: the magic, the version, zero delay and quality, the stream size and the frame count.
func (v VBRI) Bytes() []byte {
	out := binary.BigEndian.AppendUint16([]byte(vbriMagic), v.Version)

	out = append(out, make([]byte, vbriSkipped)...)
	out = binary.BigEndian.AppendUint32(out, v.Size)

	return binary.BigEndian.AppendUint32(out, v.Frames)
}
