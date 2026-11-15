// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"math/bits"
	"math/rand/v2"
)

// Layer2Subband is how a subband of a Layer II allocation table codes its samples.
type Layer2Subband struct {
	// Steps holds the steps of the class that each allocation from 1 on selects.
	Steps []int
	// Width is the width of the subband's allocation.
	Width int
}

// Layer2Audio holds the raw fields of a Layer II frame's audio data, and the allocation table they
// follow.
type Layer2Audio struct {
	// Table holds how each subband the allocation table codes takes its samples, from subband 0.
	Table []Layer2Subband
	// Allocation holds each channel's allocation for each subband. From the bound on, only the first
	// channel's is written.
	Allocation [2][Subbands]uint32
	// Selections holds each channel's scalefactor selection for each subband, written where the
	// channel's allocation is not 0.
	Selections [2][Subbands]uint32
	// Scalefactors holds each channel's scalefactor index for each subband in each part. Only those
	// of the parts the selection does not join to an earlier part are written.
	Scalefactors [2][Subbands][Layer2Parts]uint32
	// Samples holds each set's sample codes for each channel and subband, written where the
	// allocation is not 0. From the bound on, only the first channel's are written.
	Samples [Layer2Sets][2][Subbands]uint32
}

// Layer II layout.
const (
	// Layer2Parts is the number of parts of a Layer II frame, each with its own scalefactors.
	Layer2Parts = 3
	// Layer2Sets is the number of sets of subband samples in a Layer II frame.
	Layer2Sets = 36
	// granuleSets is the number of sets whose codes a granule writes together.
	granuleSets = 3
	// selectionBits is the width of a scalefactor selection.
	selectionBits = 2
	// selections is the number of scalefactor selections.
	selections = 4
)

// Scalefactor selections that join parts.
const (
	// joinSecond joins the second part to the first.
	joinSecond = 1 + iota
	// joinAll joins every part to the first.
	joinAll
	// joinThird joins the third part to the second.
	joinThird
)

// Layer II classes.
var (
	// groupedSteps holds the steps of the classes that write three samples' codes as one word.
	groupedSteps = map[int]bool{3: true, 5: true, 9: true}
	// sentParts lists, for each scalefactor selection, the parts whose scalefactors are written.
	// Selection 1 joins the second part to the first, 2 joins every part to the first, and 3 joins
	// the third part to the second.
	sentParts = [selections][]int{{0, 1, 2}, {0, 2}, {0}, {0, 1}}
)

// Layer2Audio encodes a Layer II frame's audio data in the layout of the header's mode.
//
// In joint stereo, the subbands from 4 times one more than the mode extension on carry one
// allocation and one set of codes for both channels. A class of 3, 5 or 9 steps writes a
// granule's three codes c0, c1 and c2 as the word c0 + c1 N + c2 N², in the fewest bits that hold
// N³ words. Every other class has 2^n-1 steps and writes each code in n bits.
//
// Parameters:
//   - fields: the fields.
//
// Returns:
//   - []byte: the audio data, padded with zeros to a whole byte.
func (h Header) Layer2Audio(fields *Layer2Audio) []byte {
	lay := h.layout(&fields.Allocation, len(fields.Table))

	var writer BitWriter

	lay.each(func(ch, sb int) {
		if lay.carried(ch, sb) {
			writer.Write(fields.Allocation[ch][sb], fields.Table[sb].Width)
		}
	})

	lay.each(func(ch, sb int) {
		if lay.allocation(ch, sb) != 0 {
			writer.Write(fields.Selections[ch][sb], selectionBits)
		}
	})

	lay.each(func(ch, sb int) {
		if lay.allocation(ch, sb) != 0 {
			for _, part := range sentParts[fields.Selections[ch][sb]%selections] {
				writer.Write(fields.Scalefactors[ch][sb][part], scalefactorBits)
			}
		}
	})

	for first := 0; first < Layer2Sets; first += granuleSets {
		lay.each(func(ch, sb int) {
			if allocation := lay.allocation(ch, sb); allocation != 0 && lay.carried(ch, sb) {
				writer.codes(fields.Table[sb].Steps[allocation-1], fields.Samples[first:first+granuleSets], ch, sb)
			}
		})
	}

	return writer.Bytes()
}

// RandomLayer2Audio returns Layer II audio data with random fields that follow an allocation table.
//
// Each allocation is any the subband's width holds. Each scalefactor selection is any of the four,
// and the parts it joins share their scalefactor index, which is at most 62, the largest an encoder
// writes. Each sample code is one of the steps of the channel's class.
//
// Parameters:
//   - random: the source of the fields.
//   - table: how each subband the allocation table codes takes its samples.
//
// Returns:
//   - Layer2Audio: the fields.
func RandomLayer2Audio(random *rand.Rand, table []Layer2Subband) Layer2Audio {
	var fields Layer2Audio

	fields.Table = table

	for ch := range fields.Allocation {
		for sb, subband := range table {
			allocation := random.Uint32N(1 << subband.Width)
			selection := random.Uint32N(selections)

			fields.Allocation[ch][sb], fields.Selections[ch][sb] = allocation, selection

			factors := &fields.Scalefactors[ch][sb]
			for part := range factors {
				factors[part] = random.Uint32N(largestScalefactor + 1)
			}

			switch selection {
			case joinSecond:
				factors[1] = factors[0]
			case joinAll:
				factors[1], factors[2] = factors[0], factors[0]
			case joinThird:
				factors[2] = factors[1]
			default:
			}

			if allocation == 0 {
				continue
			}

			steps := uint32(subband.Steps[allocation-1]) //nolint:gosec // Steps are at most 65535.

			for set := range Layer2Sets {
				fields.Samples[set][ch][sb] = random.Uint32N(steps)
			}
		}
	}

	return fields
}

// codes writes the codes of three samples of one subband and channel.
//
// Parameters:
//   - steps: the steps of the subband's class.
//   - sets: the granule's three sets of codes.
//   - ch: the channel.
//   - sb: the subband.
func (w *BitWriter) codes(steps int, sets [][2][Subbands]uint32, ch, sb int) {
	if !groupedSteps[steps] {
		for set := range sets {
			w.Write(sets[set][ch][sb], bits.Len(uint(steps)))
		}

		return
	}

	n := uint32(steps) //nolint:gosec // Grouped classes have at most 9 steps.
	word := sets[0][ch][sb] + n*sets[1][ch][sb] + n*n*sets[2][ch][sb]

	w.Write(word, bits.Len(uint(steps*steps*steps-1)))
}
