// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

// Sizes of the transforms.
const (
	// longTerms is n for the IMDCT of a long block, 18 lines into 36 samples.
	longTerms = Samples / 2
	// shortTerms is n for the IMDCT of a short block, 6 lines into 12 samples.
	shortTerms = shortPoints / 4
	// longPairs is the number of pairs of values the long block's DCTs of n points fold, (n-1)/2.
	longPairs = (longTerms - 1) / 2
	// shortPairs is the number of pairs of values the short block's DCTs of n points fold.
	shortPairs = (shortTerms - 1) / 2
)

// longBlock windows the 36-point IMDCT of a long block subband.
//
// Sample i of the IMDCT is the sum over the 18 lines X(k) of X(k) cos(π/72 (2i+19)(2k+1)).
//
// Parameters:
//   - dst: the windowed block.
//   - in: the subband's 18 lines.
//   - window: the block's window.
func longBlock(dst *[longPoints]float64, in []float64, window Window) {
	shape := &longWindows[window]

	imdct36(dst, (*[Samples]float64)(in))

	for at := range dst {
		dst[at] = float64(dst[at] * shape[at])
	}
}

// shortBlock windows the three 12-point IMDCTs of a short block subband and overlaps them.
//
// Sample i of each window's IMDCT is the sum over its 6 lines X(k) of X(k) cos(π/24 (2i+7)(2k+1)).
// The windows start 6, 12 and 18 samples into the block, and the block is 0 outside them.
//
// Parameters:
//   - dst: the windowed block.
//   - in: the subband's 18 lines, line f of window w at 3f+w.
func shortBlock(dst *[longPoints]float64, in []float64) {
	*dst = [longPoints]float64{}
	in = in[:Samples]

	var (
		lines   [shortPoints / 2]float64
		samples [shortPoints]float64
	)

	for window := range shortWindows {
		start := shortStep * (window + 1)

		for k := range lines {
			lines[k] = in[shortWindows*k+window]
		}

		imdct12(&samples, &lines)

		for at, sample := range samples {
			dst[start+at] += float64(sample * shortWindow[at])
		}
	}
}
