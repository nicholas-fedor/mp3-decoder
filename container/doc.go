// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

// Package container parses the tags around and inside an MPEG audio stream.
//
// It imports only [github.com/nicholas-fedor/mp3-decoder/frame], so a program that reads a stream's
// tags links no decoder.
//
// # Finding the Audio
//
// [TagSize] gives the size of an ID3v2, ID3v1 or APEv2 tag at the start of some bytes. Such a tag
// may come before the first frame or between frames. [TrailerSize] gives the size of the ID3v1,
// APEv2 and Lyrics3 v2.00 tags at a stream's end. [ID3v2Size] reads the header of an ID3v2 tag
// alone.
//
// # Reading the Tag Frame
//
// The first frame of a Layer III stream may hold a Xing, Info or VBRI tag in place of audio.
// [ParseInfo] reads it into an [Info]. [Info.Samples] gives the number of samples the stream
// decodes to, less the encoder delay and padding when a LAME tag gives them. [Info.OffsetAt] turns
// a share of the stream's length into a byte offset through the tag's seek table.
package container
