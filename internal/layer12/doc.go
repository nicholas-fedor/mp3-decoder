// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

// Package layer12 decodes the audio data of MPEG audio Layer I and Layer II frames.
//
// A [Decoder] reads each frame's bit allocation and scalefactors, then requantizes its samples and
// passes them to the synthesis filterbank. The two layers share the requantization rule and the
// scalefactor table. Layer II adds the allocation tables that its bitrate and sampling frequency
// choose, and the grouping of three samples in one code word.
package layer12
