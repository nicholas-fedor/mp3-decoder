// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// Layer II bitstream.
const (
	// layer2Parts is the number of parts of a Layer II frame, each with its own scalefactors.
	layer2Parts = 3
	// partGranules is the number of granules in a part.
	partGranules = 4
	// granuleSets is the number of sets of subband samples in a granule, which share their code
	// words.
	granuleSets = 3
	// selectionBits is the width of a subband's scalefactor selection.
	selectionBits = 2
)

// Scalefactor selections, which say which parts of a Layer II frame share a subband's scalefactors.
const (
	// eachPart gives each part its own scalefactor.
	eachPart = iota
	// firstPair gives the first two parts one scalefactor, and the third its own.
	firstPair
	// allParts gives the three parts one scalefactor.
	allParts
	// lastPair gives the first part its own scalefactor, and the last two one.
	lastPair
)

// decodeLayer2 decodes the audio of one Layer II frame, one granule of three sets at a time.
//
// The allocation table follows the frame's version, sampling frequency and bitrate per channel,
// and the subbands above those it codes hold no samples.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs.
//   - head: the frame's header.
//   - data: the frame's bytes from its header to its end.
//
// Returns:
//   - bool: whether the frame's CRC matches its bit allocation and scalefactor selections, or the
//     frame has no CRC.
func (d *Decoder) decodeLayer2(dst [][2]float64, head frame.Header, data []byte) bool {
	start, channels, bound := d.stream.Pos(), head.Channels(), jointBound(head)

	limit := d.readLayer2Allocation(allocationFor(head), channels, bound)
	d.readSelections(limit, channels)

	intact := head.CheckCRC(data, d.stream.Pos()-start)

	d.synth.Follow(channels)
	d.readLayer2Scalefactors(limit, channels)

	d.samples = [granuleSets][2][synth.Subbands]float64{}

	for part := range layer2Parts {
		d.partSteps(part, limit, channels)

		for granule := range partGranules {
			d.readGranule(limit, channels, bound)
			d.synthesize(dst[(part*partGranules+granule)*granuleSets*synth.Subbands:], granuleSets, channels)
		}
	}

	return intact
}

// readLayer2Allocation reads the class of each subband's samples from an allocation table.
//
// Below the bound each channel has its own allocation. From the bound on, both channels share one.
// An allocation of 0 gives a subband no samples.
//
// Parameters:
//   - table: the frame's allocation table.
//   - channels: the frame's number of channels.
//   - bound: the first subband whose samples both channels share.
//
// Returns:
//   - int: the number of subbands the table codes.
func (d *Decoder) readLayer2Allocation(table allocationTable, channels, bound int) int {
	sb := 0

	for _, run := range table {
		for range run.subbands {
			for ch := range channels {
				if sb >= bound && ch > 0 {
					d.classes[ch][sb] = d.classes[0][sb]

					continue
				}

				d.classes[ch][sb] = quantClass{steps: 0, bits: 0, grouped: false}
				if allocation := d.stream.Read(run.width); allocation > 0 {
					d.classes[ch][sb] = run.classes[allocation-1]
				}
			}

			sb++
		}
	}

	return sb
}

// readSelections reads the scalefactor selection of each subband with samples.
//
// Parameters:
//   - limit: the number of subbands the frame codes.
//   - channels: the frame's number of channels.
func (d *Decoder) readSelections(limit, channels int) {
	for sb := range limit {
		for ch := range channels {
			if d.classes[ch][sb].steps > 0 {
				d.selections[ch][sb] = d.stream.Read(selectionBits)
			}
		}
	}
}

// readLayer2Scalefactors reads the scalefactor indices of each subband with samples, one for each
// part its selection does not share.
//
// Parameters:
//   - limit: the number of subbands the frame codes.
//   - channels: the frame's number of channels.
func (d *Decoder) readLayer2Scalefactors(limit, channels int) {
	for sb := range limit {
		for ch := range channels {
			if d.classes[ch][sb].steps == 0 {
				continue
			}

			factors := &d.factors[ch][sb]

			factors[0] = d.stream.Read(scalefactorBits)

			switch d.selections[ch][sb] {
			case firstPair:
				factors[1], factors[2] = factors[0], d.stream.Read(scalefactorBits)
			case allParts:
				factors[1], factors[2] = factors[0], factors[0]
			case lastPair:
				factors[1] = d.stream.Read(scalefactorBits)
				factors[2] = factors[1]
			default:
				factors[1] = d.stream.Read(scalefactorBits)
				factors[2] = d.stream.Read(scalefactorBits)
			}
		}
	}
}

// partSteps works out the size of the quantization steps of each subband with samples in one part.
//
// Parameters:
//   - part: the part.
//   - limit: the number of subbands the frame codes.
//   - channels: the frame's number of channels.
func (d *Decoder) partSteps(part, limit, channels int) {
	for sb := range limit {
		for ch := range channels {
			if class := d.classes[ch][sb]; class.steps > 0 {
				d.steps[ch][sb] = class.step(scalefactors[d.factors[ch][sb][part]])
			}
		}
	}
}

// readGranule reads and requantizes the three sets of subband samples of one granule.
//
// From the bound on, both channels take the codes a subband holds, each scaled by its own
// scalefactor. A grouped code word that holds more than three samples' codes, which no encoder
// writes, gives the three samples 0.
//
// Parameters:
//   - limit: the number of subbands the frame codes.
//   - channels: the frame's number of channels.
//   - bound: the first subband whose samples both channels share.
func (d *Decoder) readGranule(limit, channels, bound int) {
	for sb := range limit {
		var (
			codes [granuleSets]int
			valid bool
		)

		for ch := range channels {
			class := d.classes[ch][sb]

			if class.steps > 0 && (sb < bound || ch == 0) {
				codes, valid = d.readCodes(class)
			}

			for set := range granuleSets {
				d.samples[set][ch][sb] = 0
				if class.steps > 0 && valid {
					d.samples[set][ch][sb] = class.requantize(codes[set], d.steps[ch][sb])
				}
			}
		}
	}
}

// readCodes reads the codes of a subband's three samples in one granule.
//
// A grouped class codes them in one word, the first in its lowest place.
//
// Parameters:
//   - class: the subband's class.
//
// Returns:
//   - [granuleSets]int: the codes.
//   - bool: false for a grouped word that holds more than three samples' codes.
func (d *Decoder) readCodes(class quantClass) ([granuleSets]int, bool) {
	if !class.grouped {
		return [granuleSets]int{
			int(d.stream.Read(class.bits)), int(d.stream.Read(class.bits)), int(d.stream.Read(class.bits)),
		}, true
	}

	word, steps := int(d.stream.Read(class.bits)), class.steps
	if word >= steps*steps*steps {
		return [granuleSets]int{}, false
	}

	return [granuleSets]int{word % steps, word / steps % steps, word / (steps * steps)}, true
}
