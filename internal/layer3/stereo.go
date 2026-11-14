// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"math"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

//go:generate go test -run ^TestStereoTables$ -update

// jointStereo restores the left and right lines of a joint stereo granule band by band.
type jointStereo struct {
	// left holds the left channel's lines.
	left *[granuleLines]float64
	// right holds the right channel's lines.
	right *[granuleLines]float64
	// factors holds the right channel's scalefactors, which give the intensity positions.
	factors *Scalefactors
	// midSide reports whether the frame codes bands with mid/side stereo.
	midSide bool
}

// Joint stereo modes, as bits of a Layer III frame's mode extension.
const (
	// intensityStereo turns on intensity stereo.
	intensityStereo = 1 << iota
	// midSideStereo turns on mid/side stereo.
	midSideStereo
)

// Joint stereo constants.
const (
	// noPosition is the lowest intensity position that codes no position.
	noPosition = 7
	// lastLongPosition is the highest long band that sends an intensity position.
	lastLongPosition = longBands - 2
	// lastShortPosition is the highest short band that sends an intensity position.
	lastShortPosition = shortBands - 2
	// invSqrt2 is 1/√2, which mid/side stereo scales by.
	invSqrt2 = math.Sqrt2 / 2
)

// Stereo turns the lines of an MPEG-1 joint stereo granule into its left and right channels.
//
// It changes nothing in a frame that is not joint stereo. Mid/side stereo turns mid and side lines
// M and S into left (M+S)/√2 and right (M-S)/√2.
//
// Intensity stereo codes the bands above the highest band in which the right channel holds a
// value. Short blocks find that band in each window. Mixed blocks code their long bands only when
// no window holds a value in a short band. A band above that bound holds the left channel's value v
// and takes its intensity position p from the right channel's scalefactors. It becomes left
// v k/(1+k) and right v/(1+k), where k = tan(p π/12). The highest long band and the highest short
// band take the position of the band below them, and a position of 7 or more codes none.
//
// The bands below the bound, and the bands that code no position, take mid/side stereo when the
// frame has it too.
//
// Parameters:
//   - head: the frame's header.
//   - left: the left channel's lines, as [Spectrum.Requantize] gives them.
//   - right: the right channel's lines, as [Spectrum.Requantize] gives them.
//   - info: the right channel's side information.
//   - factors: the right channel's scalefactors.
func Stereo(head frame.Header, left, right *[granuleLines]float64, info *GranuleInfo, factors *Scalefactors) {
	if head.Mode != frame.JointStereo {
		return
	}

	joint := jointStereo{left: left, right: right, factors: factors, midSide: head.Extension&midSideStereo != 0}

	if head.Extension&intensityStereo == 0 {
		joint.band(0, granuleLines, noPosition)

		return
	}

	partition := bandsAt(head.SampleRate)

	switch {
	case info.BlockType != ShortBlocks:
		joint.long(partition, granuleLines)
	case info.MixedBlock:
		end := windows * partition.short[mixedShortBand]

		if joint.short(partition, mixedShortBand) {
			joint.long(partition, end)
		} else {
			joint.band(0, end, noPosition)
		}
	default:
		joint.short(partition, 0)
	}
}

// band restores a run of lines that share an intensity position.
//
// Parameters:
//   - from: the first line.
//   - to: the line after the last.
//   - position: the intensity position, or 7 or more where intensity stereo does not code the run.
func (j *jointStereo) band(from, to, position int) {
	left := j.left[from:to]
	right := j.right[from:to]

	right = right[:len(left)]

	switch {
	case position < noPosition:
		toLeft, toRight := intensityRatios[position], intensityRatios[noPosition-1-position]

		for line, value := range left {
			left[line] = float64(value * toLeft)
			right[line] = float64(value * toRight)
		}
	case j.midSide:
		for line, mid := range left {
			side := right[line]

			left[line] = float64((mid + side) * invSqrt2)
			right[line] = float64((mid - side) * invSqrt2)
		}
	default:
	}
}

// long restores the long bands below a line, from the highest down.
//
// Intensity stereo codes the bands above the highest band in which the right channel holds a value.
//
// Parameters:
//   - partition: the bands of the frame's sampling frequency.
//   - end: the line the long bands end at.
func (j *jointStereo) long(partition *bands, end int) {
	coded := true

	for band := longBands - 1; band >= 0; band-- {
		from, to := partition.long[band], partition.long[band+1]
		if to > end {
			continue
		}

		coded = coded && silent(j.right[from:to])

		position := noPosition
		if coded {
			position = j.factors.Long(min(band, lastLongPosition))
		}

		j.band(from, to, position)
	}
}

// short restores each window of the short bands from a band up, from the highest band down.
//
// Intensity stereo codes the bands of each window above the highest band in which the right
// channel holds a value in that window.
//
// Parameters:
//   - partition: the bands of the frame's sampling frequency.
//   - from: the first band.
//
// Returns:
//   - bool: true when intensity stereo codes every band of every window.
func (j *jointStereo) short(partition *bands, from int) bool {
	all := true

	for window := range windows {
		coded := true

		for band := shortBands - 1; band >= from; band-- {
			width := partition.short[band+1] - partition.short[band]
			start := windows*partition.short[band] + window*width

			coded = coded && silent(j.right[start:start+width])

			position := noPosition
			if coded {
				position = j.factors.Short(min(band, lastShortPosition), window)
			}

			j.band(start, start+width, position)
		}

		all = all && coded
	}

	return all
}

// silent reports whether every line of a run is 0.
//
// Parameters:
//   - lines: the run.
//
// Returns:
//   - bool: true when every line is 0.
func silent(lines []float64) bool {
	for _, value := range lines {
		if value != 0 {
			return false
		}
	}

	return true
}
