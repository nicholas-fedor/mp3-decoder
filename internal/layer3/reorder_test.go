// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// TestReorder checks reordered granules of every kind of block at every sampling frequency.
//
// Each line starts as its own index. A short block line of window w that lies f lines into its
// window must end at 3f+w, found from the published band widths, and every other line must stay.
// The lines must end as a permutation of where they started.
func TestReorder(t *testing.T) {
	t.Parallel()

	kinds := []GranuleInfo{
		{},
		{BlockType: StartBlock},
		{BlockType: StopBlock},
		{BlockType: ShortBlocks},
		{BlockType: ShortBlocks, MixedBlock: true},
	}

	for _, widths := range publishedWidths {
		for _, kind := range kinds {
			name := fmt.Sprintf("%d Hz %+v", widths.rate, kind)
			places := linePlaces(&kind, widths.long, widths.short)

			var lines, want, identity [granuleLines]float64

			for line := range lines {
				lines[line] = float64(line)
				want[line] = float64(line)
				identity[line] = float64(line)
			}

			firsts := [shortBands]int{}
			for band := 1; band < shortBands; band++ {
				firsts[band] = firsts[band-1] + widths.short[band-1]
			}

			windowStart := 0

			for line, place := range places {
				if !place.short {
					continue
				}

				if line == 0 || !places[line-1].short || places[line-1] != place {
					windowStart = line
				}

				want[3*(firsts[place.band]+line-windowStart)+place.window] = float64(line)
			}

			Reorder(frame.Header{SampleRate: widths.rate}, &lines, &kind)

			assert.Equal(t, want, lines, name)
			assert.Equal(t, identity[:], slices.Sorted(slices.Values(lines[:])), name+" is a permutation")
		}
	}
}
