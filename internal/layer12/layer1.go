// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12

import "github.com/nicholas-fedor/mp3-decoder/internal/synth"

// Layer I bitstream.
const (
	// layer1Sets is the number of sets of subband samples in a Layer I frame.
	layer1Sets = 12
	// allocationBits is the width of a Layer I bit allocation.
	allocationBits = 4
	// forbiddenAllocation is the bit allocation no encoder writes.
	forbiddenAllocation = 15
	// scalefactorBits is the width of a scalefactor index.
	scalefactorBits = 6
	// fullRange is the width of the range from -1 to 1 that a subband's quantization steps divide.
	fullRange = 2
)

// readAllocation reads the number of bits of each subband's samples.
//
// Below the bound each channel has its own allocation. From the bound on, both channels share one.
// An allocation of 0 gives a subband no samples, and any other allocation n gives samples of n+1
// bits.
//
// Parameters:
//   - channels: the frame's number of channels.
//   - bound: the first subband whose samples both channels share.
//
// Returns:
//   - bool: false when a subband has the forbidden allocation.
func (d *Decoder) readAllocation(channels, bound int) bool {
	allowed := true

	for sb := range synth.Subbands {
		for ch := range channels {
			if sb < bound || ch == 0 {
				allocation := int(d.stream.Read(allocationBits))

				allowed = allowed && allocation != forbiddenAllocation

				d.widths[ch][sb] = 0
				if allocation > 0 {
					d.widths[ch][sb] = allocation + 1
				}
			} else {
				d.widths[ch][sb] = d.widths[0][sb]
			}
		}
	}

	return allowed
}

// readScalefactors reads the scalefactor of each subband with samples, and works out the size of
// its quantization steps.
//
// A sample of n bits with value s stands for 2(s+1-2^(n-1))/(2^n-1) times the scalefactor, so a
// step is 2/(2^n-1) times the scalefactor.
//
// Parameters:
//   - channels: the frame's number of channels.
func (d *Decoder) readScalefactors(channels int) {
	for sb := range synth.Subbands {
		for ch := range channels {
			width := d.widths[ch][sb]
			if width == 0 {
				d.steps[ch][sb] = 0

				continue
			}

			factor := scalefactors[d.stream.Read(scalefactorBits)]

			d.steps[ch][sb] = float64(factor * (fullRange / float64(int(1)<<width-1)))
		}
	}
}

// readSet reads and requantizes one set of subband samples.
//
// From the bound on, both channels take the one sample a subband holds, each scaled by its own
// scalefactor.
//
// Parameters:
//   - channels: the frame's number of channels.
//   - bound: the first subband whose samples both channels share.
func (d *Decoder) readSet(channels, bound int) {
	for sb := range synth.Subbands {
		var code int

		for ch := range channels {
			width := d.widths[ch][sb]
			if width == 0 {
				d.samples[ch][sb] = 0

				continue
			}

			if sb < bound || ch == 0 {
				code = int(d.stream.Read(width)) + 1 - 1<<(width-1)
			}

			d.samples[ch][sb] = float64(d.steps[ch][sb] * float64(code))
		}
	}
}
