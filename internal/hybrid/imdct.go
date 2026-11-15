// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

// cosineTransform holds the tables of an IMDCT of 2n lines into 4n samples, for an odd n.
type cosineTransform struct {
	// scale holds 1/(2 cos(π(2k+1)/(8n))) for each of the 2n lines.
	scale []float64
	// split holds 1/(2 cos(π(2k+1)/(4n))) for each of the n differences.
	split []float64
	// cosines holds cos(π m(2k+1)/(2n)) of the DCT of n points, at n*m+k.
	cosines []float64
}

// Sizes of the transforms.
const (
	// longTerms is n for the IMDCT of a long block, 18 lines into 36 samples.
	longTerms = Samples / 2
	// shortTerms is n for the IMDCT of a short block, 6 lines into 12 samples.
	shortTerms = shortPoints / 4
)

// Transforms of each block size.
var (
	// longTransform is the IMDCT of a long block.
	longTransform = cosineTransform{scale: longScale[:], split: longSplit[:], cosines: longCosines[:]}
	// shortTransform is the IMDCT of a short block.
	shortTransform = cosineTransform{scale: shortScale[:], split: shortSplit[:], cosines: shortCosines[:]}
)

// imdct turns 2n lines into the 4n samples of their IMDCT.
//
// Sample i is the sum over the lines X(k) of X(k) cos(π/(8n) (2i+1+2n)(2k+1)). Those angles fold
// onto the DCT-IV of the lines, c(j), the sum over k of X(k) cos(π/(8n) (2j+1)(2k+1)): sample i is
// c(n+i) below n, -c(3n-1-i) below 3n, and -c(i-3n) after.
//
// The DCT-IV comes from a DCT-II. Scaled by 1/(2 cos(π(2k+1)/(8n))), the lines give a DCT-II D
// with c(j) = D(j) + D(j+1), and D(2n) taken as 0. Byeong Gi Lee's split then turns the DCT-II of
// 2n points into two of n points: of the sums y(k) + y(2n-1-k) for the even outputs, and of the
// differences scaled by 1/(2 cos(π(2k+1)/(4n))) for the odd outputs, which add two neighbors.
//
// Parameters:
//   - dst: the 4n samples.
//   - lines: the 2n lines.
func (t *cosineTransform) imdct(dst, lines []float64) {
	points := len(lines)
	terms := points >> 1

	var sums, differences [longTerms]float64

	for k := range terms {
		low := float64(lines[k] * t.scale[k])
		high := float64(lines[points-1-k] * t.scale[points-1-k])

		sums[k] = low + high
		differences[k] = float64((low - high) * t.split[k])
	}

	// dct holds the DCT-II D, with D(2n) left at 0.
	var dct [Samples + 1]float64

	for output := range terms {
		var even, odd float64

		for k, cosine := range t.cosines[terms*output : terms*(output+1)] {
			even += float64(sums[k] * cosine)
			odd += float64(differences[k] * cosine)
		}

		dct[2*output] = even

		dct[2*output+1] += odd

		if output > 0 {
			dct[2*output-1] += odd
		}
	}

	// mirror is the first sample past the samples that count the DCT-IV down.
	mirror := points + terms

	for i := range terms {
		dst[i] = dct[terms+i] + dct[terms+i+1]
	}

	for i := terms; i < mirror; i++ {
		dst[i] = -(dct[mirror-1-i] + dct[mirror-i])
	}

	for i := mirror; i < 2*points; i++ {
		dst[i] = -(dct[i-mirror] + dct[i-mirror+1])
	}
}

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

	longTransform.imdct(dst[:], in[:Samples])

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

		shortTransform.imdct(samples[:], lines[:])

		for at, sample := range samples {
			dst[start+at] += float64(sample * shortWindow[at])
		}
	}
}
