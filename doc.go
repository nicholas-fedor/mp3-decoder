// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mp3 is a pure-Go MPEG audio decoder.
//
// It decodes streams of Layer I, Layer II and Layer III frames of MPEG-1, MPEG-2 and MPEG-2.5 into
// PCM samples. The samples are the same, bit for bit, on every architecture.
//
// # Opening a Stream
//
// [OpenStream] decodes any [io.Reader], such as a network stream. [Open] takes an [io.ReadSeeker]
// and also finds the number of samples the file decodes to, so it can seek. Both step over the tags
// at the start and find the first frame that the header after it confirms. Neither closes its
// source.
//
// # Reading Samples
//
// [Decoder.Read] fills a buffer with left and right pairs, unclipped, with full scale at 1. A mono
// stream gives each sample to both channels. Read returns an unwrapped [io.EOF] after the last
// sample. [Decoder.Info] gives the sampling frequency and the number of channels.
//
// When a Layer III stream's LAME tag gives the encoder delay and padding, the samples the encoder
// added are left out, so the tracks of an album join without a gap. Lengths and positions count
// only the samples that remain.
//
// # Seeking
//
// [Decoder.Len] gives the number of samples, and [Decoder.Seek] moves to any of them. A seek gives
// the same samples a decode from the start gives. [Decoder.Position] gives the number of the sample
// Read gives next. A decoder from OpenStream cannot seek, and its Seek returns [ErrNotSeekable].
//
// # Damage
//
// Damage never ends a stream. Bytes that are not a frame of the stream are passed over. A frame that
// fails to decode gives silence, so every frame keeps its place. [Decoder.Stats] counts what
// decoding carried on past.
//
// # Parsing Without Decoding
//
// The packages [github.com/nicholas-fedor/mp3-decoder/frame] and
// [github.com/nicholas-fedor/mp3-decoder/container] read a stream's structure without decoding its
// audio. Neither imports the decoder.
package mp3
