// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package layer3 decodes the audio data of MPEG audio Layer III frames.
//
// A [Decoder] takes each frame in turn. It reads the side information ([SideInfo]), then takes the
// frame's main data from the bit reservoir ([Reservoir]). Each granule's scalefactors and Huffman
// data then give its requantized spectral values. [Stereo] restores them to left and right, and
// [Reorder] puts the lines of short blocks in frequency order. The hybrid and synthesis
// filterbanks then turn them into PCM samples.
//
// A frame whose main data begins before the bytes the reservoir holds gives silence. It still
// feeds the reservoir and the filterbanks, so the frames after it decode as they would in the
// whole stream.
package layer3
