// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer3

import "github.com/nicholas-fedor/mp3-decoder/frame"

// Reorder puts the short block lines of a granule in order of frequency, the three windows
// interleaved.
//
// Short blocks hold each short band's lines window by window. Reorder moves line f of window w,
// counted from the window's first line, to line 3f+w. Mixed blocks keep their long block lines in
// place, and a granule of long blocks is left as it is.
//
// Parameters:
//   - head: the frame's header.
//   - lines: the channel's lines, as [Stereo] or [Spectrum.Requantize] gives them.
//   - info: the channel's side information.
func Reorder(head frame.Header, lines *[granuleLines]float64, info *GranuleInfo) {
	if info.BlockType != ShortBlocks {
		return
	}

	partition := bandsAt(head.SampleRate)

	from := 0
	if info.MixedBlock {
		from = mixedShortBand
	}

	var held [granuleLines]float64

	start := windows * partition.short[from]
	copy(held[start:], lines[start:])

	for band := from; band < shortBands; band++ {
		first := partition.short[band]
		width := partition.short[band+1] - first

		for window := range windows {
			at := windows*first + window*width

			for line, value := range held[at : at+width] {
				lines[windows*(first+line)+window] = value
			}
		}
	}
}
