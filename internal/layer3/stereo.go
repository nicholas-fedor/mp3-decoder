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
	// scale is the lower-rate intensity scale, 0 for a ratio of 2^(-1/4) and 1 for 2^(-1/2).
	scale int
	// midSide reports whether the frame codes bands with mid/side stereo.
	midSide bool
	// lowerRate reports whether the frame is MPEG-2 or MPEG-2.5, whose positions take the
	// lower-rate ratios.
	lowerRate bool
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
	// noPosition is the lowest MPEG-1 intensity position that codes no position.
	noPosition = 7
	// uncoded stands for a band that intensity stereo does not code.
	uncoded = -1
	// lastLongPosition is the highest long band that sends an intensity position.
	lastLongPosition = longBands - 2
	// lastShortPosition is the highest short band that sends an intensity position.
	lastShortPosition = shortBands - 2
	// invSqrt2 is 1/√2, which mid/side stereo scales by.
	invSqrt2 = math.Sqrt2 / 2
)

// Stereo turns the lines of a joint stereo granule into its left and right channels.
//
// It changes nothing in a frame that is not joint stereo. Mid/side stereo turns mid and side lines
// M and S into left (M+S)/√2 and right (M-S)/√2.
//
// Intensity stereo codes the bands above the highest band in which the right channel holds a
// value. Short blocks find that band in each window. Mixed blocks code their long bands only when
// no window holds a value in a short band. A band above that bound holds the left channel's value v
// and takes its intensity position p from the right channel's scalefactors.
//
// In MPEG-1 a band becomes left v k/(1+k) and right v/(1+k), where k = tan(p π/12), and a position
// of 7 or more codes none. The highest long band and the highest short band take the position of
// the band below them.
//
// At the lower sampling frequencies the right channel's scalefac_compress gives an intensity ratio
// r of 2^(-1/4), or 2^(-1/2) when it is odd. An odd position scales the left channel by
// r^((p+1)/2), an even one scales the right channel by r^(p/2), and the largest value a
// scalefactor's width holds codes none. The highest long band and the highest short band take the
// position of the band below them when intensity stereo codes that band too, and position 0
// otherwise.
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

	joint := jointStereo{
		left: left, right: right, factors: factors, scale: info.ScalefacCompress & 1,
		midSide: head.Extension&midSideStereo != 0, lowerRate: head.Version != frame.MPEG1,
	}

	if !usesIntensity(head) {
		joint.band(0, granuleLines, uncoded)

		return
	}

	partition := bandsAt(head.SampleRate)

	switch {
	case info.BlockType != ShortBlocks:
		joint.long(partition, granuleLines)
	case info.MixedBlock:
		end := partition.mixedEnd()

		if joint.short(partition, mixedShortBand) {
			joint.long(partition, end)
		} else {
			joint.band(0, end, uncoded)
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
//   - position: the intensity position, or -1 where intensity stereo does not code the run.
func (j *jointStereo) band(from, to, position int) {
	left := j.left[from:to]
	right := j.right[from:to]

	right = right[:len(left)]

	switch {
	case position != uncoded:
		toLeft, toRight := j.ratios(position)

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

		position := uncoded
		if coded {
			position = j.longPosition(partition, band)
		}

		j.band(from, to, position)
	}
}

// longPosition returns the intensity position of a long band that intensity stereo codes.
//
// Parameters:
//   - partition: the bands of the frame's sampling frequency.
//   - band: the long band.
//
// Returns:
//   - int: the position, or -1 where the band codes none.
func (j *jointStereo) longPosition(partition *bands, band int) int {
	if band <= lastLongPosition {
		return j.position(j.factors.Long(band), j.factors.widestLong(band))
	}

	if j.lowerRate && !silent(j.right[partition.long[band-1]:partition.long[band]]) {
		return 0
	}

	return j.longPosition(partition, band-1)
}

// position returns the intensity position a scalefactor gives.
//
// Parameters:
//   - value: the scalefactor.
//   - widest: the largest value its width holds.
//
// Returns:
//   - int: the position, or -1 where it codes none, which a value of 7 or more does in MPEG-1 and
//     the largest value its width holds does at the lower sampling frequencies.
func (j *jointStereo) position(value, widest int) int {
	if j.lowerRate && value >= widest || !j.lowerRate && value >= noPosition {
		return uncoded
	}

	return value
}

// ratios returns how an intensity position scales the left channel's value into each channel.
//
// Parameters:
//   - position: the position.
//
// Returns:
//   - toLeft: the scale of the left channel.
//   - toRight: the scale of the right channel.
//
//nolint:nonamedreturns // Same-type returns need names.
func (j *jointStereo) ratios(position int) (toLeft, toRight float64) {
	if !j.lowerRate {
		return intensityRatios[position], intensityRatios[noPosition-1-position]
	}

	steps := (j.scale + 1) * ((position + 1) >> 1)
	if position&1 == 1 {
		return gain(-steps), 1
	}

	return 1, gain(-steps)
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

			position := uncoded
			if coded {
				position = j.shortPosition(partition, band, window)
			}

			j.band(start, start+width, position)
		}

		all = all && coded
	}

	return all
}

// shortPosition returns the intensity position of a short band that intensity stereo codes in
// one window.
//
// Parameters:
//   - partition: the bands of the frame's sampling frequency.
//   - band: the short band.
//   - window: the window.
//
// Returns:
//   - int: the position, or -1 where the band codes none.
func (j *jointStereo) shortPosition(partition *bands, band, window int) int {
	if band <= lastShortPosition {
		return j.position(j.factors.Short(band, window), j.factors.widestShort(band, window))
	}

	width := partition.short[band] - partition.short[band-1]
	start := windows*partition.short[band-1] + window*width

	if j.lowerRate && !silent(j.right[start:start+width]) {
		return 0
	}

	return j.shortPosition(partition, band-1, window)
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

// usesIntensity reports whether a frame codes intensity stereo, a joint stereo frame with the
// intensity bit of its mode extension set.
//
// Parameters:
//   - head: the frame's header.
//
// Returns:
//   - bool: true for intensity stereo.
func usesIntensity(head frame.Header) bool {
	return head.Mode == frame.JointStereo && head.Extension&intensityStereo != 0
}
