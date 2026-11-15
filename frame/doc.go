// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package frame parses MPEG audio frame headers.
//
// It reads the headers of MPEG-1, MPEG-2 and MPEG-2.5 frames in Layers I, II and III. It imports
// nothing of the decoder, so a program that needs only a stream's format or its frames can use it
// alone.
//
// # Headers
//
// [Parse] reads a header into a [Header], and rejects the reserved and forbidden values the frame
// search depends on. A Header gives the frame's [Version], [Layer], [Mode], sampling frequency and
// bitrate. From those it gives the frame's size ([Header.Size]), its samples per channel
// ([Header.Samples]) and its channels ([Header.Channels]). [Header.SameStream] reports whether two
// headers can belong to one stream.
//
// # Finding Frames
//
// Bytes inside a frame or in damage can look like a header. [Find] therefore finds the first frame
// that the header after it confirms. A free-format frame needs two headers after it at one spacing, which gives its
// bitrate.
//
// # Checking and Seeking
//
// [Header.CheckCRC] checks the CRC of a protected frame. [Header.Preroll] gives how many frames
// before a frame decoding must start for that frame to decode exactly. A player that opens a stream
// at a byte offset starts that many frames earlier.
package frame
