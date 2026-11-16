// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package hybrid

// aliasButterflies is the number of butterflies across each subband boundary.
const aliasButterflies = 8

// reduceAliases undoes the encoder's alias reduction across each boundary between two of the
// lowest subbands.
//
// Each boundary takes 8 butterflies, between the 8 lines on either side. A butterfly turns the
// lower line l and upper line u into l Cs - u Ca and u Cs + l Ca.
//
// Parameters:
//   - lines: the granule's lines.
//   - long: the number of subbands, from the lowest, that reduction runs between.
func reduceAliases(lines *[granuleLines]float64, long int) {
	for boundary := 1; boundary < long; boundary++ {
		at := boundary * Samples

		for i := range aliasButterflies {
			lower, upper := lines[at-1-i], lines[at+i]

			lines[at-1-i] = float64(lower*aliasCs[i]) - float64(upper*aliasCa[i])
			lines[at+i] = float64(upper*aliasCs[i]) + float64(lower*aliasCa[i])
		}
	}
}
