// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import "encoding/binary"

// Version is an MPEG audio version.
type Version uint8

// Layer is an MPEG audio layer.
type Layer uint8

// Mode is a channel mode.
type Mode uint8

// Header is a parsed frame header.
type Header struct {
	// SampleRate is the sampling frequency in Hz.
	SampleRate int
	// Bitrate is the bitrate in bit/s, or 0 for a free-format header that Find has not measured.
	Bitrate int
	// Version is the MPEG version.
	Version Version
	// Layer is the layer.
	Layer Layer
	// Mode is the channel mode.
	Mode Mode
	// Extension is the mode extension of a joint stereo frame.
	Extension uint8
	// Free reports whether the stream is free format, whose bitrate the header does not give.
	Free bool
	// Padding reports whether the frame holds one extra slot.
	Padding bool
	// Protected reports whether a CRC follows the header.
	Protected bool
}

// MPEG versions.
const (
	// MPEG1 is MPEG-1 audio.
	MPEG1 Version = iota + 1
	// MPEG2 is MPEG-2 audio at the lower sampling frequencies.
	MPEG2
	// MPEG25 is the MPEG-2.5 extension of Layer III.
	MPEG25
)

// Layers.
const (
	// Layer1 is Layer I.
	Layer1 Layer = iota + 1
	// Layer2 is Layer II.
	Layer2
	// Layer3 is Layer III.
	Layer3
)

// Channel modes.
const (
	// Stereo is two independent channels.
	Stereo Mode = iota + 1
	// JointStereo is two channels coded together.
	JointStereo
	// DualChannel is two separate programs.
	DualChannel
	// Mono is one channel.
	Mono
)

// Sizes of the parts that open a frame.
const (
	// HeaderSize is the size of a frame header in bytes.
	HeaderSize = 4
	// crcSize is the size of the CRC that follows a protected header in bytes.
	crcSize = 2
)

// MaxSize is the size in bytes of the largest frame.
//
// Only a free-format frame reaches it. The largest frame at a fixed bitrate holds 1729 bytes.
const MaxSize = 3456

// Header fields, as shifts of the 32-bit header.
const (
	// syncShift is the shift of the 11-bit sync word.
	syncShift = 21
	// versionShift is the shift of the two version bits.
	versionShift = 19
	// layerShift is the shift of the layer.
	layerShift = 17
	// protectionShift is the shift of the protection bit.
	protectionShift = 16
	// bitrateShift is the shift of the bitrate index.
	bitrateShift = 12
	// rateShift is the shift of the sampling frequency index.
	rateShift = 10
	// paddingShift is the shift of the padding bit.
	paddingShift = 9
	// modeShift is the shift of the mode.
	modeShift = 6
	// extensionShift is the shift of the mode extension.
	extensionShift = 4
)

// Header field values.
const (
	// syncWord is the sync word.
	syncWord = 0x7ff
	// twoBits masks a two-bit field.
	twoBits = 0x3
	// fourBits masks a four-bit field.
	fourBits = 0xf
	// rateReserved is the reserved sampling frequency index.
	rateReserved = 3
	// bitrateForbidden is the forbidden bitrate index.
	bitrateForbidden = 15
)

// Frame lengths.
const (
	// layer1Samples is the number of samples in a Layer I frame.
	layer1Samples = 384
	// longSamples is the number of samples in a Layer II frame and an MPEG-1 Layer III frame.
	longSamples = 1152
	// shortSamples is the number of samples in an MPEG-2 or MPEG-2.5 Layer III frame.
	shortSamples = 576
	// layer1Slot is the size of a Layer I slot in bytes.
	layer1Slot = 4
	// bitsPerByte is the number of bits in a byte.
	bitsPerByte = 8
	// stereoChannels is the number of channels in every mode but mono.
	stereoChannels = 2
)

// Layer III side information sizes in bytes.
const (
	// sideMPEG1Mono is the side information size of a mono MPEG-1 frame.
	sideMPEG1Mono = 17
	// sideMPEG1Stereo is the side information size of a two-channel MPEG-1 frame.
	sideMPEG1Stereo = 32
	// sideLSFMono is the side information size of a mono MPEG-2 or MPEG-2.5 frame.
	sideLSFMono = 9
	// sideLSFStereo is the side information size of a two-channel MPEG-2 or MPEG-2.5 frame.
	sideLSFStereo = 17
)

// Parse parses the frame header at the start of head.
//
// It rejects only reserved and forbidden field values. A header alone does not prove a frame, since
// audio data can hold a sync word by chance.
//
// Parameters:
//   - head: the bytes, of which the first four are read.
//
// Returns:
//   - Header: the header.
//   - bool: false when head does not start with a valid header.
func Parse(head []byte) (Header, bool) {
	var none Header

	if len(head) < HeaderSize {
		return none, false
	}

	word := binary.BigEndian.Uint32(head)
	version := versions[word>>versionShift&twoBits]
	layer := layers[word>>layerShift&twoBits]
	index := word >> bitrateShift & fourBits
	rate := word >> rateShift & twoBits

	if word>>syncShift != syncWord || version == 0 || layer == 0 {
		return none, false
	}

	if index == bitrateForbidden || rate == rateReserved {
		return none, false
	}

	return Header{
		SampleRate: sampleRates[version][rate],
		Bitrate:    bitrate(version, layer, index),
		Version:    version,
		Layer:      layer,
		Mode:       modes[word>>modeShift&twoBits],
		Extension:  uint8(word >> extensionShift & twoBits),
		Free:       index == 0,
		Padding:    word>>paddingShift&1 == 1,
		Protected:  word>>protectionShift&1 == 0,
	}, true
}

// Channels returns the number of channels.
//
// Returns:
//   - int: 1 for mono, 2 otherwise.
func (h Header) Channels() int {
	if h.Mode == Mono {
		return 1
	}

	return stereoChannels
}

// DataAt returns the offset from the frame's start of the data after the header, the CRC and the
// Layer III side information.
//
// Returns:
//   - int: the offset.
func (h Header) DataAt() int {
	at := HeaderSize + h.SideInfoSize()
	if h.Protected {
		at += crcSize
	}

	return at
}

// SameStream reports whether two headers can belong to the same stream.
//
// A stream keeps its version, layer and sampling frequency, and a free-format stream stays free
// format. Its bitrate, padding and mode can change from frame to frame.
//
// Parameters:
//   - other: the other header.
//
// Returns:
//   - bool: whether the headers can belong to the same stream.
func (h Header) SameStream(other Header) bool {
	return h.Version == other.Version && h.Layer == other.Layer && h.SampleRate == other.SampleRate &&
		h.Free == other.Free
}

// Samples returns the number of samples per channel in the frame.
//
// Returns:
//   - int: 384 for Layer I, 576 for MPEG-2 and MPEG-2.5 Layer III, 1152 otherwise.
func (h Header) Samples() int {
	switch {
	case h.Layer == Layer1:
		return layer1Samples
	case h.Layer == Layer3 && h.Version != MPEG1:
		return shortSamples
	default:
		return longSamples
	}
}

// SideInfoSize returns the size of the Layer III side information.
//
// Returns:
//   - int: the size in bytes, or 0 for Layers I and II.
func (h Header) SideInfoSize() int {
	switch {
	case h.Layer != Layer3:
		return 0
	case h.Version == MPEG1 && h.Mode == Mono:
		return sideMPEG1Mono
	case h.Version == MPEG1:
		return sideMPEG1Stereo
	case h.Mode == Mono:
		return sideLSFMono
	default:
		return sideLSFStereo
	}
}

// Size returns the size of the frame, header included.
//
// A free-format header gives no size until Find measures its bitrate.
//
// Returns:
//   - int: the size in bytes, or 0 when the bitrate is unknown.
func (h Header) Size() int {
	if h.Bitrate == 0 {
		return 0
	}

	return h.Samples()*h.Bitrate/(bitsPerByte*h.slot()*h.SampleRate)*h.slot() + h.padding()
}

// padding returns the size of the frame's padding.
//
// Returns:
//   - int: the size in bytes, which is one slot or 0.
func (h Header) padding() int {
	if h.Padding {
		return h.slot()
	}

	return 0
}

// slot returns the size of one slot.
//
// Returns:
//   - int: the size in bytes, 4 for Layer I and 1 otherwise.
func (h Header) slot() int {
	if h.Layer == Layer1 {
		return layer1Slot
	}

	return 1
}

// bitrate looks up a bitrate index.
//
// MPEG-2.5 shares the MPEG-2 table.
//
// Parameters:
//   - version: the MPEG version.
//   - layer: the layer.
//   - index: the bitrate index, below 15.
//
// Returns:
//   - int: the bitrate in bit/s, or 0 for free format.
func bitrate(version Version, layer Layer, index uint32) int {
	if version == MPEG1 {
		return mpeg1Bitrates[layer][index] * kilo
	}

	return lsfBitrates[layer][index] * kilo
}
