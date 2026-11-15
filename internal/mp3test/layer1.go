// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import "math/rand/v2"

// Layer1Audio holds the raw fields of a Layer I frame's audio data.
type Layer1Audio struct {
	// Allocation holds each channel's bit allocation for each subband. From the bound on, only the
	// first channel's is written.
	Allocation [2][Subbands]uint32
	// Scalefactors holds each channel's scalefactor index for each subband, written where the
	// channel's allocation is not 0.
	Scalefactors [2][Subbands]uint32
	// Samples holds each set's sample codes for each channel and subband, written where the
	// allocation is not 0. From the bound on, only the first channel's are written.
	Samples [Layer1Sets][2][Subbands]uint32
}

// audioLayout is the layout of a Layer I or Layer II frame's audio data in one mode.
type audioLayout struct {
	// allocations holds the allocations written.
	allocations *[2][Subbands]uint32
	// channels is the number of channels.
	channels int
	// bound is the first subband whose allocation and samples both channels share.
	bound int
	// subbands is the number of subbands the frame codes.
	subbands int
}

// Layer I layout.
const (
	// Subbands is the number of subbands.
	Subbands = 32
	// Layer1Sets is the number of sets of subband samples in a Layer I frame.
	Layer1Sets = 12
	// Layer1AllocationBits is the width of a Layer I bit allocation.
	Layer1AllocationBits = 4
	// scalefactorBits is the width of a scalefactor index.
	scalefactorBits = 6
	// largestScalefactor is the largest scalefactor index an encoder writes.
	largestScalefactor = 62
	// boundStep is the number of subbands each step of the mode extension moves the bound by.
	boundStep = 4
)

// Layer1Audio encodes a Layer I frame's audio data in the layout of the header's mode.
//
// In joint stereo, the subbands from 4 times one more than the mode extension on carry one
// allocation and one sample for both channels. A subband whose allocation is n carries samples of
// n+1 bits.
//
// Parameters:
//   - fields: the fields.
//
// Returns:
//   - []byte: the audio data, padded with zeros to a whole byte.
func (h Header) Layer1Audio(fields *Layer1Audio) []byte {
	lay := h.layout(&fields.Allocation, Subbands)

	var writer BitWriter

	lay.each(func(ch, sb int) {
		if lay.carried(ch, sb) {
			writer.Write(fields.Allocation[ch][sb], Layer1AllocationBits)
		}
	})

	lay.each(func(ch, sb int) {
		if lay.allocation(ch, sb) != 0 {
			writer.Write(fields.Scalefactors[ch][sb], scalefactorBits)
		}
	})

	for set := range Layer1Sets {
		lay.each(func(ch, sb int) {
			if allocation := lay.allocation(ch, sb); allocation != 0 && lay.carried(ch, sb) {
				writer.Write(fields.Samples[set][ch][sb], int(allocation)+1)
			}
		})
	}

	return writer.Bytes()
}

// layout returns the layout of a frame's audio data in the header's mode.
//
// Parameters:
//   - allocations: the allocations written.
//   - subbands: the number of subbands the frame codes.
//
// Returns:
//   - audioLayout: the layout.
func (h Header) layout(allocations *[2][Subbands]uint32, subbands int) audioLayout {
	lay := audioLayout{allocations: allocations, channels: len(allocations), bound: Subbands, subbands: subbands}

	switch h.Mode {
	case Mono:
		lay.channels = 1
	case JointStereo:
		lay.bound = boundStep * (int(h.Extension) + 1)
	default:
	}

	return lay
}

// allocation returns the allocation a channel takes for a subband, the first channel's from the
// bound on.
//
// Parameters:
//   - ch: the channel.
//   - sb: the subband.
//
// Returns:
//   - uint32: the allocation.
func (l audioLayout) allocation(ch, sb int) uint32 {
	if sb < l.bound {
		return l.allocations[ch][sb]
	}

	return l.allocations[0][sb]
}

// carried reports whether the frame carries a channel's own allocation and samples for a
// subband, which it does below the bound and for the first channel from it on.
//
// Parameters:
//   - ch: the channel.
//   - sb: the subband.
//
// Returns:
//   - bool: whether the frame carries them.
func (l audioLayout) carried(ch, sb int) bool {
	return sb < l.bound || ch == 0
}

// each calls a function for each subband the frame codes, and within it for each channel.
//
// Parameters:
//   - visit: the function, given the channel and the subband.
func (l audioLayout) each(visit func(ch, sb int)) {
	for sb := range l.subbands {
		for ch := range l.channels {
			visit(ch, sb)
		}
	}
}

// RandomLayer1Audio returns Layer I audio data with random fields.
//
// Each allocation is at most a limit. Each scalefactor index is at most 62, the largest an encoder
// writes. Each sample code fits its subband's allocation, short of all ones, which no encoder
// writes.
//
// Parameters:
//   - random: the source of the fields.
//   - most: the largest allocation, below 15.
//
// Returns:
//   - Layer1Audio: the fields.
func RandomLayer1Audio(random *rand.Rand, most uint32) Layer1Audio {
	var fields Layer1Audio

	for ch := range fields.Allocation {
		for sb := range Subbands {
			allocation := random.Uint32N(most + 1)

			fields.Allocation[ch][sb] = allocation
			fields.Scalefactors[ch][sb] = random.Uint32N(largestScalefactor + 1)

			for set := range Layer1Sets {
				fields.Samples[set][ch][sb] = random.Uint32N(1<<(allocation+1) - 1)
			}
		}
	}

	return fields
}
