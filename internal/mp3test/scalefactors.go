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

// lsfRange is a range of lower-rate scalefac_compress values.
//
// A value less the range's first value holds the widths of the four partitions as the digits of a
// mixed-radix number, partition 0 the most significant. Partitions 1 to 3 have the radices radices
// gives, where 0 sends no scalefactors of width above 0.
type lsfRange struct {
	// radices holds the radix of the widths of partitions 1 to 3.
	radices [3]int
	// first is the range's first value.
	first int
	// table is the row of the partition counts the range takes.
	table int
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
	// lsfMixedLong is the number of long bands lower-rate mixed blocks write.
	lsfMixedLong = 6
	// lsfMixedBlocks is the column of lsfCounts that mixed blocks take.
	lsfMixedBlocks = 2
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

// Ranges of lower-rate scalefac_compress values, each with its radices, first value and table.
var (
	// lsfRanges holds the ranges of every channel but the right channel of an intensity stereo
	// frame.
	lsfRanges = [...]lsfRange{{[3]int{5, 4, 4}, 0, 0}, {[3]int{5, 4, 0}, 400, 1}, {[3]int{3, 0, 0}, 500, 2}}
	// intensityRanges holds the ranges of the right channel of an intensity stereo frame, whose
	// scalefac_compress is halved.
	intensityRanges = [...]lsfRange{{[3]int{6, 6, 0}, 0, 3}, {[3]int{4, 4, 0}, 180, 4}, {[3]int{3, 0, 0}, 244, 5}}
)

// lsfCounts holds how many scalefactors each of the four partitions of lower-rate scalefactors
// writes, at [table][blocks][partition], where blocks is 0 for long blocks, 1 for short blocks
// and 2 for mixed blocks.
var lsfCounts = [6][3][4]int{
	{{6, 5, 5, 5}, {9, 9, 9, 9}, {6, 9, 9, 9}},
	{{6, 5, 7, 3}, {9, 9, 12, 6}, {6, 9, 12, 6}},
	{{11, 10, 0, 0}, {18, 18, 0, 0}, {15, 18, 0, 0}},
	{{7, 7, 7, 0}, {12, 12, 12, 0}, {6, 15, 12, 0}},
	{{6, 6, 6, 3}, {12, 9, 9, 6}, {6, 12, 9, 6}},
	{{8, 8, 5, 0}, {15, 12, 9, 0}, {6, 18, 9, 0}},
}

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

// LSFScalefactors writes the MPEG-2 or MPEG-2.5 scalefactors of one granule of any channel but the
// right channel of an intensity stereo frame.
//
// scalefac_compress gives the widths of four partitions and the table of their counts. Each
// scalefactor keeps the low bits its width holds, in factors too.
//
// Parameters:
//   - granule: the granule's side information.
//   - factors: the scalefactors.
func (w *BitWriter) LSFScalefactors(granule *Granule, factors *Scalefactors) {
	w.writeLSF(granule, lsfRanges[:], int(granule.ScalefacCompress), factors)
}

// IntensityScalefactors writes the MPEG-2 or MPEG-2.5 scalefactors of one granule of the right
// channel of an intensity stereo frame, whose scalefac_compress halved gives the widths and the
// table.
//
// Each scalefactor keeps the low bits its width holds, in factors too.
//
// Parameters:
//   - granule: the granule's side information.
//   - factors: the scalefactors.
func (w *BitWriter) IntensityScalefactors(granule *Granule, factors *Scalefactors) {
	w.writeLSF(granule, intensityRanges[:], int(granule.ScalefacCompress>>1), factors)
}

// partitions writes runs of scalefactors, each run of a partition's count at its width.
//
// Parameters:
//   - slots: the scalefactors in the order they are sent, each cut to its width.
//   - counts: the count of each partition.
//   - widths: the width of each partition.
func (w *BitWriter) partitions(slots []*uint32, counts, widths [4]int) {
	at := 0

	for partition, count := range counts {
		for _, slot := range slots[at : at+count] {
			*slot &= 1<<widths[partition] - 1
			w.Write(*slot, widths[partition])
		}

		at += count
	}
}

// writeLSF writes the partitions of lower-rate scalefactors in the order they are sent: long bands
// 0 to 20, each window of short bands 0 to 11, or long bands 0 to 5 then each window of short bands
// 3 to 11.
//
// Parameters:
//   - granule: the granule's side information.
//   - ranges: the ranges of scalefac_compress values the channel takes.
//   - compress: the value, halved for the right channel of an intensity stereo frame.
//   - factors: the scalefactors, cut to their widths.
func (w *BitWriter) writeLSF(granule *Granule, ranges []lsfRange, compress int, factors *Scalefactors) {
	var picked lsfRange

	for _, candidate := range ranges {
		if compress >= candidate.first {
			picked = candidate
		}
	}

	widths := picked.widths(compress)

	var slots []*uint32

	if granule.WindowSwitching && granule.BlockType == shortBlocks {
		blocks, from := 1, 0

		if granule.MixedBlock {
			blocks, from = lsfMixedBlocks, mixedShort

			for band := range lsfMixedLong {
				slots = append(slots, &factors.Long[band])
			}
		}

		for band := from; band < len(factors.Short); band++ {
			for window := range factors.Short[band] {
				slots = append(slots, &factors.Short[band][window])
			}
		}

		w.partitions(slots, lsfCounts[picked.table][blocks], widths)

		return
	}

	for band := range factors.Long {
		slots = append(slots, &factors.Long[band])
	}

	w.partitions(slots, lsfCounts[picked.table][0], widths)
}

// widths returns the width of each partition for a value in the range, taking the digits from the
// most significant down.
//
// Parameters:
//   - compress: the value.
//
// Returns:
//   - [4]int: the widths of partitions 0 to 3.
func (r lsfRange) widths(compress int) [4]int {
	var widths [4]int

	rest := compress - r.first

	for partition := range widths {
		if partition > 0 && r.radices[partition-1] == 0 {
			continue
		}

		place := 1
		for _, radix := range r.radices[partition:] {
			place *= max(radix, 1)
		}

		widths[partition], rest = rest/place, rest%place
	}

	return widths
}
