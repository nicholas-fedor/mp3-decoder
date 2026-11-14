// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import "encoding/binary"

// Header holds the raw field values of a frame header.
type Header struct {
	// Version is the two version bits.
	Version uint32
	// Layer is the two layer bits.
	Layer uint32
	// Bitrate is the bitrate index.
	Bitrate uint32
	// Rate is the sampling frequency index.
	Rate uint32
	// Mode is the two mode bits.
	Mode uint32
	// Extension is the mode extension.
	Extension uint32
	// Emphasis is the two emphasis bits.
	Emphasis uint32
	// Padding sets the padding bit.
	Padding bool
	// Protected clears the protection bit, so a CRC follows the header.
	Protected bool
}

// Version bits.
const (
	// MPEG25 is the version bits of MPEG-2.5.
	MPEG25 = 0
	// MPEG2 is the version bits of MPEG-2.
	MPEG2 = 2
	// MPEG1 is the version bits of MPEG-1.
	MPEG1 = 3
)

// Layer bits.
const (
	// Layer3 is the layer bits of Layer III.
	Layer3 = 1
	// Layer2 is the layer bits of Layer II.
	Layer2 = 2
	// Layer1 is the layer bits of Layer I.
	Layer1 = 3
)

// Mode bits.
const (
	// Stereo is the mode bits of stereo.
	Stereo = 0
	// JointStereo is the mode bits of joint stereo.
	JointStereo = 1
	// DualChannel is the mode bits of dual channel.
	DualChannel = 2
	// Mono is the mode bits of a single channel.
	Mono = 3
)

// Header field positions, as shifts of the 32-bit header.
const (
	// syncWord is the 11-bit sync word.
	syncWord = 0x7ff
	// syncShift is the shift of the sync word.
	syncShift = 21
	// versionShift is the shift of the version bits.
	versionShift = 19
	// layerShift is the shift of the layer bits.
	layerShift = 17
	// protectionShift is the shift of the protection bit.
	protectionShift = 16
	// bitrateShift is the shift of the bitrate index.
	bitrateShift = 12
	// rateShift is the shift of the sampling frequency index.
	rateShift = 10
	// paddingShift is the shift of the padding bit.
	paddingShift = 9
	// modeShift is the shift of the mode bits.
	modeShift = 6
	// extensionShift is the shift of the mode extension.
	extensionShift = 4
	// headerSize is the size of a frame header in bytes.
	headerSize = 4
)

// Bytes encodes the header.
//
// Returns:
//   - []byte: the four header bytes.
func (h Header) Bytes() []byte {
	word := uint32(syncWord)<<syncShift | h.Version<<versionShift | h.Layer<<layerShift |
		h.Bitrate<<bitrateShift | h.Rate<<rateShift | h.Mode<<modeShift | h.Extension<<extensionShift |
		h.Emphasis

	if !h.Protected {
		word |= 1 << protectionShift
	}

	if h.Padding {
		word |= 1 << paddingShift
	}

	return binary.BigEndian.AppendUint32(nil, word)
}

// Frame builds a frame of the header, filled with zeros.
//
// Parameters:
//   - size: the frame's size in bytes, at least the header's.
//
// Returns:
//   - []byte: the frame.
func (h Header) Frame(size int) []byte {
	return append(h.Bytes(), make([]byte, max(size, headerSize)-headerSize)...)
}

// Place copies payload into frame at an offset.
//
// Parameters:
//   - frame: the frame.
//   - at: the offset.
//   - payload: the bytes to copy, which must fit.
//
// Returns:
//   - []byte: frame.
func Place(frame []byte, at int, payload []byte) []byte {
	copy(frame[at:], payload)

	return frame
}
