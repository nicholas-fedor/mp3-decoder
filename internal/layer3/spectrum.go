// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman"
)

// Spectrum holds the quantized values of one channel's granule.
type Spectrum struct {
	// Values holds the quantized value of each of the granule's 576 lines.
	//
	// Long blocks hold the lines in order of frequency. Short blocks hold each short band's lines
	// window by window, as the bitstream sends them, and mixed blocks hold 36 long block lines
	// before their short bands.
	Values [granuleLines]int
	// Decoded is the number of values the Huffman data gave, and every value after them is 0.
	Decoded int
}

// Granule sizes.
const (
	// granuleLines is the number of lines in a granule.
	granuleLines = 576
	// pairValues is the number of values in a big_values pair.
	pairValues = 2
)

// Implicit region boundaries of a granule with window switching.
const (
	// shortRegion1 is the short band region1 starts at in short blocks, counted in each window.
	shortRegion1 = 3
	// longRegion1 is the long band region1 starts at in start and stop blocks.
	longRegion1 = 8
)

// Read decodes the quantized values of one channel's granule from its Huffman data.
//
// The big_values pairs come first, in up to three regions that each have a table, then count1
// quadruples until the granule's bits end or fewer than 4 lines remain. Decoding stops at the end
// of the granule's bits. A pair that starts before the end is decoded whole, and a quadruple that
// runs past it is left out.
//
// Parameters:
//   - stream: the main data, at the granule's Huffman data after its scalefactors.
//   - head: the frame's header, as [frame.Parse] gives it.
//   - info: the granule's side information.
//   - end: the position of the bit after the granule's part2_3_length bits.
func (s *Spectrum) Read(stream *bits.Reader, head frame.Header, info *GranuleInfo, end int) {
	region1, region2 := regions(info, bandsAt(head.SampleRate))
	lines := pairValues * info.BigValues
	starts := [4]int{0, min(region1, lines), min(region2, lines), lines}

	decoded := 0

	for region, table := range info.TableSelect {
		from, to := starts[region], starts[region+1]

		decoded = from + huffman.BigValues(stream, table, s.Values[from:to], end)
		if decoded < to {
			break
		}
	}

	decoded += huffman.Count1(stream, info.Count1TableSelect, s.Values[decoded:], end)

	clear(s.Values[decoded:])

	s.Decoded = decoded
}

// regions returns the first lines of region1 and region2 of a granule's big_values pairs.
//
// A granule without window switching gives its regions in scalefactor bands. In a granule with
// window switching, region1 starts after short bands 0 to 2 of each window in short blocks and
// after long bands 0 to 7 in start and stop blocks, and runs to the end.
//
// Parameters:
//   - info: the granule's side information.
//   - partition: the bands of the frame's sampling frequency.
//
// Returns:
//   - region1: the first line of region1.
//   - region2: the first line of region2.
//
//nolint:nonamedreturns // Same-type returns need names.
func regions(info *GranuleInfo, partition *bands) (region1, region2 int) {
	switch info.BlockType {
	case NormalBlock:
		first := info.Region0Count + 1
		second := min(first+info.Region1Count+1, longBands)

		return partition.long[first], partition.long[second]
	case ShortBlocks:
		return windows * partition.short[shortRegion1], granuleLines
	default:
		return partition.long[longRegion1], granuleLines
	}
}
