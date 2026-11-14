// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package trig computes the sines and cosines that the decoder's generated tables hold, each
// rounded to the nearest float64.
//
// Only the tests that generate tables use it, so no table depends on the accuracy of package math.
package trig
