// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container

import (
	"encoding/binary"
	"slices"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// Info is what a stream's first frame says about the stream in a Xing, Info or VBRI tag.
type Info struct {
	// TOC is the seek table of a Xing or Info tag, or nil.
	//
	// Entry i is where the stream reaches i percent of its length, in 256ths of its size.
	TOC []byte
	// Frames is the number of audio frames after the tag's frame, or 0 when the tag does not say.
	Frames int64
	// Bytes is the size of the stream from the tag's frame on, or 0 when the tag does not say.
	Bytes int64
	// Delay is the number of samples the encoder added at the start.
	Delay int
	// Padding is the number of samples the encoder added at the end.
	Padding int
	// Gapless reports whether the tag gives Delay and Padding.
	Gapless bool
}

// Xing and Info tags.
const (
	// xingMagic starts the tag of a variable bitrate stream.
	xingMagic = "Xing"
	// infoMagic starts the same tag in a constant bitrate stream.
	infoMagic = "Info"
	// fieldSize is the size of the magic, the flags and each count.
	fieldSize = 4
	// hasFrames flags the frame count.
	hasFrames = 0x1
	// hasBytes flags the stream size.
	hasBytes = 0x2
	// hasTOC flags the seek table.
	hasTOC = 0x4
	// hasQuality flags the quality indicator.
	hasQuality = 0x8
	// tocSize is the number of entries in the seek table.
	tocSize = 100
	// tocScale is the unit of a seek table entry, in parts of the stream's size.
	tocScale = 256
	// tocTop is the highest percent read from the seek table, just short of its end.
	tocTop = 99.99
)

// The LAME extension after the Xing or Info fields.
const (
	// lameSize is the size of the part of the extension read.
	lameSize = 24
	// lameDelayAt is the offset of the encoder delay and padding, two 12-bit values.
	lameDelayAt = 21
	// nibbleBits is the number of bits in half a byte.
	nibbleBits = 4
	// byteBits is the number of bits in a byte.
	byteBits = 8
	// nibble masks half a byte.
	nibble = 0xf
)

// VBRI tags.
const (
	// vbriMagic starts a VBRI tag.
	vbriMagic = "VBRI"
	// vbriAt is the offset of the tag from the frame's start.
	vbriAt = frame.HeaderSize + 32
	// vbriVersion is the only version read.
	vbriVersion = 1
	// vbriVersionAt is the offset of the version in the tag.
	vbriVersionAt = 4
	// vbriBytesAt is the offset of the stream size in the tag.
	vbriBytesAt = 10
	// vbriFramesAt is the offset of the frame count in the tag.
	vbriFramesAt = 14
	// vbriSize is the size of the part of the tag read.
	vbriSize = 18
)

// ParseInfo parses the Xing, Info or VBRI tag in a stream's first frame.
//
// A Xing or Info tag follows the side information. A protected frame may also hold it after the
// CRC. A tag whose flags name fields that the frame ends before keeps the fields it holds.
//
// Parameters:
//   - head: the frame's header.
//   - data: the frame's bytes from its header onward.
//
// Returns:
//   - Info: what the tag says.
//   - bool: false when the frame holds no tag.
func ParseInfo(head frame.Header, data []byte) (Info, bool) {
	info, ok := parseXing(data, frame.HeaderSize+head.SideInfoSize())
	if !ok && head.Protected {
		info, ok = parseXing(data, head.DataAt())
	}

	if ok {
		return info, true
	}

	return parseVBRI(data)
}

// OffsetAt returns where a point of the stream lies, by the seek table.
//
// It interpolates between the entries of the percents on either side.
//
// Parameters:
//   - fraction: the point, as a share of the stream's length from 0 to 1.
//   - size: the stream's size in bytes.
//
// Returns:
//   - int64: the byte offset from the tag's frame.
//   - bool: false when there is no seek table or size is not positive.
func (i Info) OffsetAt(fraction float64, size int64) (int64, bool) {
	if len(i.TOC) != tocSize || size <= 0 {
		return 0, false
	}

	pct := min(max(fraction*tocSize, 0), tocTop)
	whole := int(pct)

	from, to := float64(i.TOC[whole]), float64(tocScale)
	if whole+1 < tocSize {
		to = float64(i.TOC[whole+1])
	}

	share := from + (to-from)*(pct-float64(whole))

	return int64(share / tocScale * float64(size)), true
}

// Samples returns the number of samples per channel the stream decodes to.
//
// With the encoder delay and padding known, it counts only the encoded audio.
//
// Parameters:
//   - head: the header of the tag's frame.
//
// Returns:
//   - int64: the number of samples, or 0 when the tag gives no frame count.
func (i Info) Samples(head frame.Header) int64 {
	total := i.Frames * int64(head.Samples())
	if i.Gapless {
		total -= int64(i.Delay + i.Padding)
	}

	return max(total, 0)
}

// parseXing parses a Xing or Info tag.
//
// Parameters:
//   - data: the frame's bytes from its header onward.
//   - at: the offset of the tag.
//
// Returns:
//   - Info: what the tag says.
//   - bool: false when no tag starts at the offset.
func parseXing(data []byte, at int) (Info, bool) {
	var info Info

	if at+2*fieldSize > len(data) {
		return info, false
	}

	magic := string(data[at : at+fieldSize])
	if magic != xingMagic && magic != infoMagic {
		return info, false
	}

	flags := binary.BigEndian.Uint32(data[at+fieldSize:])
	rest := data[at+2*fieldSize:]

	var ok bool

	info.Frames, rest, ok = field(rest, flags, hasFrames)
	if ok {
		info.Bytes, rest, ok = field(rest, flags, hasBytes)
	}

	if ok {
		info.TOC, rest, ok = table(rest, flags)
	}

	if ok {
		_, rest, ok = field(rest, flags, hasQuality)
	}

	if ok {
		info.Delay, info.Padding, info.Gapless = lame(rest)
	}

	return info, true
}

// field reads one count of a Xing or Info tag when its flag is set.
//
// Parameters:
//   - rest: the tag's bytes from the count onward.
//   - flags: the tag's flags.
//   - flag: the count's flag.
//
// Returns:
//   - int64: the count, or 0 when the flag is clear.
//   - []byte: the bytes after the count.
//   - bool: false when the flag is set and rest ends before the count does.
func field(rest []byte, flags, flag uint32) (int64, []byte, bool) {
	switch {
	case flags&flag == 0:
		return 0, rest, true
	case len(rest) < fieldSize:
		return 0, rest, false
	default:
		return int64(binary.BigEndian.Uint32(rest)), rest[fieldSize:], true
	}
}

// table reads the seek table of a Xing or Info tag when its flag is set.
//
// Parameters:
//   - rest: the tag's bytes from the table onward.
//   - flags: the tag's flags.
//
// Returns:
//   - toc: a copy of the table, or nil when the flag is clear.
//   - after: the bytes after the table.
//   - ok: false when the flag is set and rest ends before the table does.
//
//nolint:nonamedreturns // Same-type returns need names.
func table(rest []byte, flags uint32) (toc, after []byte, ok bool) {
	switch {
	case flags&hasTOC == 0:
		return nil, rest, true
	case len(rest) < tocSize:
		return nil, rest, false
	default:
		return slices.Clone(rest[:tocSize]), rest[tocSize:], true
	}
}

// lame reads the encoder delay and padding from the LAME extension.
//
// Any encoder name starts the extension, as LAME's own fields are the only ones read.
//
// Parameters:
//   - rest: the bytes after the Xing or Info fields.
//
// Returns:
//   - delay: the encoder delay in samples.
//   - padding: the padding in samples.
//   - ok: false when rest holds no extension.
//
//nolint:nonamedreturns // Same-type returns need names.
func lame(rest []byte) (delay, padding int, ok bool) {
	if len(rest) < lameSize || rest[0] == 0 {
		return 0, 0, false
	}

	delays := rest[lameDelayAt:lameSize]

	delay = int(delays[0])<<nibbleBits | int(delays[1]>>nibbleBits)
	padding = int(delays[1]&nibble)<<byteBits | int(delays[2])

	return delay, padding, true
}

// parseVBRI parses a VBRI tag.
//
// Parameters:
//   - data: the frame's bytes from its header onward.
//
// Returns:
//   - Info: the tag's frame count and stream size.
//   - bool: false when the frame holds no VBRI tag of the version read.
func parseVBRI(data []byte) (Info, bool) {
	var info Info

	if len(data) < vbriAt+vbriSize || string(data[vbriAt:vbriAt+fieldSize]) != vbriMagic {
		return info, false
	}

	tag := data[vbriAt:]
	if binary.BigEndian.Uint16(tag[vbriVersionAt:]) != vbriVersion {
		return info, false
	}

	info.Bytes = int64(binary.BigEndian.Uint32(tag[vbriBytesAt:]))
	info.Frames = int64(binary.BigEndian.Uint32(tag[vbriFramesAt:]))

	return info, true
}
