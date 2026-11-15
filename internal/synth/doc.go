// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package synth turns polyphase subband samples into PCM samples, for every layer.
//
// Each [Filterbank] turns 32 subband samples into 32 PCM samples through a fast 32-point DCT and a
// 512-tap window. It carries each set into the output of the next 15. [Filterbanks] holds one per
// channel, and restarts them when the number of channels changes.
package synth
