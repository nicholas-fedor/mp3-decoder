// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import "github.com/nicholas-fedor/mp3-decoder/internal/bits"

// Scalefactors holds the scalefactors of one channel's granule, in the order the bitstream sends
// them.
//
// Long blocks send one scalefactor for each of long bands 0 to 20. Short blocks send one for each
// window of short bands 0 to 11, a band's three windows in turn. Mixed blocks send long bands 0 to 7
// and then short bands 3 to 11. The last band of each kind has none and reads as 0.
type Scalefactors struct {
	// values holds the scalefactors in the order they are sent, then zeros for the last band.
	values [scalefactorCount]uint8
	// mixed reports whether the values are those of mixed blocks.
	mixed bool
}

// Scalefactor counts.
const (
	// scalefactorCount is the most scalefactors a granule holds: 13 short bands of 3 windows.
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
	// mixedShift is how many places before short blocks' values mixed blocks' short values sit,
	// since mixed blocks send 8 long scalefactors in place of the 9 of short bands 0 to 2.
	mixedShift = 1
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

// ReadMPEG1 reads the MPEG-1 scalefactors of one channel's granule.
//
// The second granule keeps the scalefactors of each long band group that scfsi marks, so both
// granules of a channel are read into one Scalefactors. A stream may mark groups after a first
// granule of short blocks, which it should not, and those groups then keep the values s holds in
// the same places.
//
// Parameters:
//   - stream: the main data, at the granule's scalefactors.
//   - side: the frame's side information.
//   - gr: the granule, 0 or 1.
//   - ch: the channel, 0 or 1.
func (s *Scalefactors) ReadMPEG1(stream *bits.Reader, side *SideInfo, gr, ch int) {
	info := &side.Granules[gr][ch]
	slen := slens[info.ScalefacCompress]

	if info.BlockType != ShortBlocks {
		s.mixed = false

		for group, scfsi := range side.SCFSI[ch] {
			if gr == 1 && scfsi {
				continue
			}

			s.read(stream, scfsiGroups[group], scfsiGroups[group+1], slen[group/2])
		}

		s.values[longBands-1] = 0

		return
	}

	s.mixed = info.MixedBlock

	shift := 0
	if s.mixed {
		shift = mixedShift
	}

	s.read(stream, 0, shortFirst-shift, slen[0])
	s.read(stream, shortFirst-shift, shortSent-shift, slen[1])
	clear(s.values[shortSent-shift:])
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
	at := band*windows + window
	if s.mixed {
		at -= mixedShift
	}

	return int(s.values[at])
}

// read reads scalefactors of one width into a run of values.
//
// Parameters:
//   - stream: the main data, at the first scalefactor.
//   - from: the first value.
//   - to: the value after the last.
//   - width: the width of each scalefactor in bits, where 0 sends none and reads 0.
func (s *Scalefactors) read(stream *bits.Reader, from, to, width int) {
	for at := from; at < to; at++ {
		s.values[at] = uint8(stream.Read(width)) //nolint:gosec // A scalefactor is at most 4 bits wide.
	}
}
