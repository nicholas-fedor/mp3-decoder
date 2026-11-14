// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hybrid turns the spectral lines of a Layer III granule into polyphase subband samples.
//
// Alias reduction comes first. Each subband's block then takes an IMDCT and a window, and
// overlaps the block before it. The frequency inversion of the polyphase filterbank is undone
// last.
package hybrid
