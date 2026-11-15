// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// Scalefactors holds the scalefactors of one channel's granule, in the order the bitstream sends
// them.
//
// Long blocks send one scalefactor for each of long bands 0 to 20. Short blocks send one for each
// window of short bands 0 to 11, a band's three windows in turn. Mixed blocks send long bands 0 to
// 7, or 0 to 5 at the lower sampling frequencies, and then short bands 3 to 11. The last band of
// each kind has none and reads as 0.
type Scalefactors struct {
	// shift is how many places before their places in short blocks the short values sit.
	//
	// It is 0 for long and short blocks. For mixed blocks it is the 9 short values of bands 0 to 2
	// less the long values sent in their place.
	shift int
	// values holds the scalefactors in the order they are sent, then zeros for the last band.
	values [scalefactorCount]uint8
	// preflag reports whether the pretab adds to the long bands.
	preflag bool
}

// lsfSplit is how a range of lower-rate scalefac_compress values gives the widths of the four
// partitions of a granule's scalefactors.
//
// A value less the range's first is a mixed-radix number. Its last digit, in radix radices[2],
// is the width of partition 3, and so on down, and what is left is the width of partition 0. A
// radix of 0 gives a width of 0.
type lsfSplit struct {
	// radices holds the radix of the widths of partitions 1 to 3.
	radices [3]int
	// first is the range's first value.
	first int
	// table is the row of the partition counts the range takes.
	table int
	// preflag reports whether the range sets preflag.
	preflag bool
}

// Scalefactor counts.
const (
	// scalefactorCount is the most scalefactors a granule holds, 3 windows of 13 short bands.
	scalefactorCount = 39
	// longBands is the number of long bands.
	longBands = 22
	// windows is the number of short windows in a granule.
	windows = 3
	// shortSent is the number of short scalefactors sent, 3 windows of short bands 0 to 11.
	shortSent = 36
	// shortFirst is the number of short scalefactors that take slen1, 3 windows of short bands 0
	// to 5.
	shortFirst = 18
	// mpeg1MixedShift is how many places before short blocks' values the short values of MPEG-1
	// mixed blocks sit, since they send 8 long scalefactors in place of the 9 of short bands 0 to 2.
	mpeg1MixedShift = 1
	// lsfMixedShift is how many places before short blocks' values the short values of lower-rate
	// mixed blocks sit, since they send 6 long scalefactors in place of the 9 of short bands 0 to 2.
	lsfMixedShift = 3
	// lsfPartitions is the number of partitions of lower-rate scalefactors.
	lsfPartitions = 4
)

// scfsiGroups holds the first long band of each scfsi group, then the band after the last.
//
// Groups 0 and 1 take slen1, and groups 2 and 3 take slen2.
var scfsiGroups = [5]int{0, 6, 11, 16, 21}

// slens holds the widths of the scalefactors at each scalefac_compress, slen1 then slen2.
var slens = [16][2]int{
	{0, 0},
	{0, 1},
	{0, 2},
	{0, 3},
	{3, 0},
	{1, 1},
	{1, 2},
	{1, 3},
	{2, 1},
	{2, 2},
	{2, 3},
	{3, 1},
	{3, 2},
	{3, 3},
	{4, 2},
	{4, 3},
}

// Splits of lower-rate scalefac_compress values.
var (
	// lsfSplits splits the values of every channel but the right channel of an intensity stereo
	// frame.
	//
	// Each split gives its radices, first value, row and preflag.
	lsfSplits = [...]lsfSplit{
		{[3]int{5, 4, 4}, 0, 0, false},
		{[3]int{5, 4, 0}, 400, 1, false},
		{[3]int{3, 0, 0}, 500, 2, true},
	}
	// intensitySplits splits the values of the right channel of an intensity stereo frame, halved
	// past the lowest bit, which is the intensity scale.
	intensitySplits = [...]lsfSplit{
		{[3]int{6, 6, 0}, 0, 3, false},
		{[3]int{4, 4, 0}, 180, 4, false},
		{[3]int{3, 0, 0}, 244, 5, false},
	}
)

// lsfCounts holds how many scalefactors each partition of lower-rate scalefactors sends, at
// [table][blocks][partition], where blocks is 0 for long blocks, 1 for short blocks and 2 for mixed
// blocks.
var lsfCounts = [6][3][lsfPartitions]int{
	{{6, 5, 5, 5}, {9, 9, 9, 9}, {6, 9, 9, 9}},
	{{6, 5, 7, 3}, {9, 9, 12, 6}, {6, 9, 12, 6}},
	{{11, 10, 0, 0}, {18, 18, 0, 0}, {15, 18, 0, 0}},
	{{7, 7, 7, 0}, {12, 12, 12, 0}, {6, 15, 12, 0}},
	{{6, 6, 6, 3}, {12, 9, 9, 6}, {6, 12, 9, 6}},
	{{8, 8, 5, 0}, {15, 12, 9, 0}, {6, 18, 9, 0}},
}

// Long returns the scalefactor of a long band.
//
// Parameters:
//   - band: the long band, from 0 to 21, or to 7 in mixed blocks.
//
// Returns:
//   - int: the scalefactor.
func (s *Scalefactors) Long(band int) int {
	return int(s.values[band])
}

// Read reads the scalefactors of one channel's granule.
//
// In MPEG-1 the second granule keeps the scalefactors of each long band group that scfsi marks, so
// both granules of a channel are read into one Scalefactors. A stream may mark groups after a first
// granule of short blocks, which it should not, and those groups then keep the values s holds in
// the same places.
//
// At the lower sampling frequencies scalefac_compress gives the widths of four partitions of the
// scalefactors and how many each sends, and the right channel of an intensity stereo frame takes
// partitions of its own.
//
// The granule's preflag is kept with its scalefactors for requantization. At the lower sampling
// frequencies scalefac_compress gives it.
//
// Parameters:
//   - stream: the main data, at the granule's scalefactors.
//   - head: the frame's header.
//   - side: the frame's side information.
//   - gr: the granule, 0 or 1.
//   - ch: the channel, 0 or 1.
func (s *Scalefactors) Read(stream *bits.Reader, head frame.Header, side *SideInfo, gr, ch int) {
	if head.Version == frame.MPEG1 {
		s.readMPEG1(stream, side, gr, ch)
	} else {
		s.readLSF(stream, head, &side.Granules[gr][ch], ch)
	}
}

// Short returns the scalefactor of a short band in one window.
//
// Parameters:
//   - band: the short band, from 0 to 12, or from 3 in mixed blocks.
//   - window: the window, from 0 to 2.
//
// Returns:
//   - int: the scalefactor.
func (s *Scalefactors) Short(band, window int) int {
	return int(s.values[band*windows+window-s.shift])
}

// readLSF reads the lower-rate scalefactors of one channel's granule.
//
// Parameters:
//   - stream: the main data, at the granule's scalefactors.
//   - head: the frame's header.
//   - info: the granule's side information.
//   - ch: the channel, 0 or 1.
func (s *Scalefactors) readLSF(stream *bits.Reader, head frame.Header, info *GranuleInfo, ch int) {
	split, compress := splitOf(head, info, ch)
	blocks := 0

	s.shift = 0

	switch {
	case info.BlockType == ShortBlocks && info.MixedBlock:
		blocks, s.shift = 2, lsfMixedShift
	case info.BlockType == ShortBlocks:
		blocks = 1
	default:
	}

	s.preflag = split.preflag

	widths := split.widths(compress)
	at := 0

	for partition, count := range lsfCounts[split.table][blocks] {
		s.readRun(stream, at, at+count, widths[partition])

		at += count
	}

	clear(s.values[at:])
}

// readMPEG1 reads the MPEG-1 scalefactors of one channel's granule.
//
// Parameters:
//   - stream: the main data, at the granule's scalefactors.
//   - side: the frame's side information.
//   - gr: the granule, 0 or 1.
//   - ch: the channel, 0 or 1.
func (s *Scalefactors) readMPEG1(stream *bits.Reader, side *SideInfo, gr, ch int) {
	info := &side.Granules[gr][ch]
	slen := slens[info.ScalefacCompress]

	s.preflag = info.Preflag

	if info.BlockType != ShortBlocks {
		s.shift = 0

		for group, scfsi := range side.SCFSI[ch] {
			if gr == 1 && scfsi {
				continue
			}

			s.readRun(stream, scfsiGroups[group], scfsiGroups[group+1], slen[group/2])
		}

		s.values[longBands-1] = 0

		return
	}

	s.shift = 0
	if info.MixedBlock {
		s.shift = mpeg1MixedShift
	}

	s.readRun(stream, 0, shortFirst-s.shift, slen[0])
	s.readRun(stream, shortFirst-s.shift, shortSent-s.shift, slen[1])
	clear(s.values[shortSent-s.shift:])
}

// readRun reads scalefactors of one width into a run of values.
//
// Parameters:
//   - stream: the main data, at the first scalefactor.
//   - from: the first value.
//   - to: the value after the last.
//   - width: the width of each scalefactor in bits, where 0 sends none and reads 0.
func (s *Scalefactors) readRun(stream *bits.Reader, from, to, width int) {
	for at := from; at < to; at++ {
		s.values[at] = uint8(stream.Read(width)) //nolint:gosec // A scalefactor is at most 4 bits wide.
	}
}

// widths returns the width of each partition's scalefactors for a scalefac_compress value in the
// split's range.
//
// Parameters:
//   - compress: the value, halved for the right channel of an intensity stereo frame.
//
// Returns:
//   - [4]int: the widths of partitions 0 to 3.
func (l lsfSplit) widths(compress int) [lsfPartitions]int {
	var widths [lsfPartitions]int

	rest := compress - l.first

	for partition := len(l.radices); partition > 0; partition-- {
		if radix := l.radices[partition-1]; radix > 0 {
			widths[partition], rest = rest%radix, rest/radix
		}
	}

	widths[0] = rest

	return widths
}

// splitOf returns the split of a granule's lower-rate scalefac_compress, and the value it splits.
//
// The right channel of an intensity stereo frame takes splits of its own, of the value halved.
//
// Parameters:
//   - head: the frame's header.
//   - info: the granule's side information.
//   - ch: the channel, 0 or 1.
//
// Returns:
//   - lsfSplit: the split of the range the value lies in.
//   - int: the value.
func splitOf(head frame.Header, info *GranuleInfo, ch int) (lsfSplit, int) {
	compress, splits := info.ScalefacCompress, lsfSplits[:]
	if ch == 1 && usesIntensity(head) {
		compress, splits = compress>>1, intensitySplits[:]
	}

	split := splits[0]

	for _, candidate := range splits {
		if compress >= candidate.first {
			split = candidate
		}
	}

	return split, compress
}
