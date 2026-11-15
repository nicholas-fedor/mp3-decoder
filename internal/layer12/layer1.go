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
)

// readAllocation reads the class of each subband's samples.
//
// Below the bound each channel has its own allocation. From the bound on, both channels share one.
// An allocation of 0 gives a subband no samples, and any other allocation n gives samples of n+1
// bits, whose 2^(n+1)-1 steps the codes short of all ones take.
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
			if sb >= bound && ch > 0 {
				d.classes[ch][sb] = d.classes[0][sb]

				continue
			}

			allocation := int(d.stream.Read(allocationBits))

			allowed = allowed && allocation != forbiddenAllocation

			d.classes[ch][sb] = quantClass{steps: 0, bits: 0, grouped: false}
			if allocation > 0 {
				d.classes[ch][sb] = quantClass{steps: 1<<(allocation+1) - 1, bits: allocation + 1, grouped: false}
			}
		}
	}

	return allowed
}

// readScalefactors reads the scalefactor of each subband with samples, and works out the size of
// its quantization steps.
//
// Parameters:
//   - channels: the frame's number of channels.
func (d *Decoder) readScalefactors(channels int) {
	for sb := range synth.Subbands {
		for ch := range channels {
			if class := d.classes[ch][sb]; class.steps > 0 {
				d.steps[ch][sb] = class.step(scalefactors[d.stream.Read(scalefactorBits)])
			}
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
			class := d.classes[ch][sb]
			if class.steps == 0 {
				d.samples[0][ch][sb] = 0

				continue
			}

			if sb < bound || ch == 0 {
				code = int(d.stream.Read(class.bits))
			}

			d.samples[0][ch][sb] = class.requantize(code, d.steps[ch][sb])
		}
	}
}
