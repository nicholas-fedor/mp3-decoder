// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import "math"

//go:generate go test -run ^TestRequantizeTables$ -update

// Requantization constants.
const (
	// gainOffset is the global_gain whose gain is 1.
	gainOffset = 210
	// subblockStep is how many quarter steps of gain each step of subblock_gain takes away.
	subblockStep = 8
	// maxMagnitude is the largest quantized magnitude, 15 and the most 13 linbits add.
	maxMagnitude = 15 + 1<<13 - 1
	// quarterSteps is the number of quarter steps in a factor of 2.
	quarterSteps = 4
	// exponentBias is the bias of a float64's exponent.
	exponentBias = 1023
	// mantissaBits is the number of bits below a float64's exponent.
	mantissaBits = 52
)

// pretab holds how many scalefactor steps preflag adds to each long band.
var pretab = [longBands]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 3, 3, 3, 2, 0}

// Requantize scales the values Read decoded to the granule's spectral values.
//
// Each value v becomes sign(v) |v|^(4/3) 2^(e/4). The exponent e is global_gain - 210, less the
// band's scalefactor in steps of 2, or of 4 with scalefac_scale. Long bands add the pretab to their
// scalefactors when preflag is set, and short bands take away 8 for each step of the window's
// subblock_gain. Mixed blocks scale their lines as long bands up to where short band 3 starts, 36
// lines in and 72 at 8 kHz, and as short bands from there. The spectral values keep the order of
// the quantized ones.
//
// Parameters:
//   - dst: where the spectral values go.
//   - info: the granule's side information.
//   - factors: the granule's scalefactors.
func (s *Spectrum) Requantize(dst *[granuleLines]float64, info *GranuleInfo, factors *Scalefactors) {
	var at int

	switch {
	case info.BlockType != ShortBlocks:
		at = s.requantizeLong(dst, info, factors, granuleLines)
	case info.MixedBlock:
		at = s.requantizeLong(dst, info, factors, s.bands.mixedEnd())
		at = s.requantizeShort(dst, info, factors, mixedShortBand, at)
	default:
		at = s.requantizeShort(dst, info, factors, 0, 0)
	}

	clear(dst[at:])
}

// requantizeLong requantizes long bands from the first line up to an end or the last decoded
// value.
//
// Parameters:
//   - dst: where the spectral values go.
//   - info: the granule's side information.
//   - factors: the granule's scalefactors.
//   - end: the line the long bands end at.
//
// Returns:
//   - int: the line after the last band requantized.
func (s *Spectrum) requantizeLong(dst *[granuleLines]float64, info *GranuleInfo, factors *Scalefactors, end int) int {
	base := info.GlobalGain - gainOffset
	at := 0

	for band := 0; at < end && at < s.Decoded; band++ {
		factor := factors.Long(band)
		if factors.preflag {
			factor += pretab[band]
		}

		next := s.bands.long[band+1]
		s.scale(dst, at, next, base-factor<<info.scalefacShift())

		at = next
	}

	return at
}

// requantizeShort requantizes each window of short bands from a band up to the last decoded
// value.
//
// Parameters:
//   - dst: where the spectral values go.
//   - info: the granule's side information.
//   - factors: the granule's scalefactors.
//   - from: the first band.
//   - at: the line the first band starts at.
//
// Returns:
//   - int: the line after the last band requantized.
func (s *Spectrum) requantizeShort(
	dst *[granuleLines]float64, info *GranuleInfo, factors *Scalefactors, from, at int,
) int {
	base := info.GlobalGain - gainOffset

	for band := from; band < shortBands && at < s.Decoded; band++ {
		width := s.bands.short[band+1] - s.bands.short[band]

		for window, subblock := range info.SubblockGain {
			s.scale(dst, at, at+width, base-subblockStep*subblock-factors.Short(band, window)<<info.scalefacShift())

			at += width
		}
	}

	return at
}

// scale requantizes a run of values that share one exponent.
//
// Parameters:
//   - dst: where the spectral values go.
//   - from: the first line.
//   - to: the line after the last.
//   - quarters: the exponent e of the gain 2^(e/4).
func (s *Spectrum) scale(dst *[granuleLines]float64, from, to, quarters int) {
	gain := gain(quarters)

	for line := from; line < to; line++ {
		switch value := s.Values[line]; {
		case value > 0:
			dst[line] = float64(pow43[value] * gain)
		case value < 0:
			dst[line] = -float64(pow43[-value] * gain)
		default:
			dst[line] = 0
		}
	}
}

// gain returns 2^(e/4).
//
// The result is exact: 2^(r/4) for the remainder r times a power of two built from its bits. An
// exponent from the side information and scalefactors lies between -1298 and 45, so the power of
// two lies between 2^-325 and 2^11 and is never subnormal.
//
// Parameters:
//   - quarters: the exponent e.
//
// Returns:
//   - float64: the gain.
func gain(quarters int) float64 {
	biased := uint64(quarters>>2 + exponentBias) //nolint:gosec // It lies between 698 and 1034.

	return float64(quarterPowers[quarters&(quarterSteps-1)] * math.Float64frombits(biased<<mantissaBits))
}
