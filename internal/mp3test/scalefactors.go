// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

// Scalefactors holds the raw scalefactors of one granule of one channel.
type Scalefactors struct {
	// Short holds scalefac_s of each window of short bands 0 to 11, at [band][window].
	Short [12][3]uint32
	// Long holds scalefac_l of long bands 0 to 20.
	Long [21]uint32
}

// MPEG-1 scalefactor layout.
const (
	// shortBlocks is the block_type of short blocks.
	shortBlocks = 2
	// shortSecond is the first short band whose scalefactors take slen2.
	shortSecond = 6
	// mixedLong is the number of long bands mixed blocks write.
	mixedLong = 8
	// mixedShort is the first short band mixed blocks write.
	mixedShort = 3
)

// MPEG-1 scalefactor widths at each scalefac_compress.
var (
	// slen1 is the width of the scalefactors of long bands 0 to 10 and short bands 0 to 5.
	slen1 = [16]int{0, 0, 0, 0, 3, 1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4}
	// slen2 is the width of the scalefactors of long bands 11 to 20 and short bands 6 to 11.
	slen2 = [16]int{0, 1, 2, 3, 0, 1, 2, 3, 1, 2, 3, 1, 2, 3, 2, 3}
)

// longGroups holds the first long band of each scfsi group, then the band after the last.
var longGroups = [5]int{0, 6, 11, 16, 21}

// MPEG1Scalefactors writes the MPEG-1 scalefactors of one granule of one channel.
//
// Short blocks write each window of short bands 0 to 11, a band's windows in turn, and mixed blocks
// write long bands 0 to 7 before short bands 3 to 11. Other blocks write long bands 0 to 20 in their
// four scfsi groups of bands 0 to 5, 6 to 10, 11 to 15 and 16 to 20, skipping each group reuse
// marks.
//
// Parameters:
//   - granule: the granule's side information.
//   - reuse: the scfsi of each group, set only for the second granule.
//   - factors: the scalefactors, each within its width.
func (w *BitWriter) MPEG1Scalefactors(granule *Granule, reuse [4]bool, factors *Scalefactors) {
	widths := [2]int{slen1[granule.ScalefacCompress], slen2[granule.ScalefacCompress]}

	if granule.WindowSwitching && granule.BlockType == shortBlocks {
		from := 0

		if granule.MixedBlock {
			for band := range mixedLong {
				w.Write(factors.Long[band], widths[0])
			}

			from = mixedShort
		}

		w.shortScalefactors(from, widths, factors)

		return
	}

	for group, skip := range reuse {
		if skip {
			continue
		}

		for band := longGroups[group]; band < longGroups[group+1]; band++ {
			w.Write(factors.Long[band], widths[group/2])
		}
	}
}

// shortScalefactors writes the MPEG-1 scalefactors of each window of short bands from one band to
// band 11.
//
// Parameters:
//   - from: the first band.
//   - widths: slen1 and slen2.
//   - factors: the scalefactors, each within its width.
func (w *BitWriter) shortScalefactors(from int, widths [2]int, factors *Scalefactors) {
	for band := from; band < len(factors.Short); band++ {
		for _, factor := range factors.Short[band] {
			w.Write(factor, widths[band/shortSecond])
		}
	}
}
