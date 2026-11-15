// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12

import "github.com/nicholas-fedor/mp3-decoder/frame"

// quantClass is a Layer II class of quantization, the steps its samples take and how they are
// coded.
type quantClass struct {
	// steps is the number of quantization steps, which is odd, so the steps center on 0.
	steps int
	// bits is the width of a sample's code, or of the code word that holds three samples when
	// grouped.
	bits int
	// grouped reports whether three samples share one code word, the first in its lowest place.
	grouped bool
}

// allocationRun is a run of subbands of a Layer II allocation table that code their samples alike.
type allocationRun struct {
	// classes holds the quantization class that each allocation from 1 on selects.
	classes []quantClass
	// subbands is the number of subbands in the run.
	subbands int
	// width is the number of bits of each subband's allocation.
	width int
}

// allocationTable is a Layer II bit allocation table, the runs of the subbands a frame codes from
// subband 0 up.
type allocationTable []allocationRun

// The bitrates per channel, in bit/s, and the sampling frequencies that choose an MPEG-1
// allocation table.
const (
	// narrowBelow is the bitrate per channel below which a frame takes a table of 8 or 12 subbands.
	narrowBelow = 56000
	// mediumUpTo is the highest bitrate per channel at which a frame takes the table of 27
	// subbands at every sampling frequency.
	mediumUpTo = 80000
	// wideRate is the sampling frequency, in Hz, whose frames take the table of 27 subbands at
	// every higher bitrate.
	wideRate = 48000
	// narrowRate is the sampling frequency, in Hz, whose frames take the table of 12 subbands at the
	// lowest bitrates.
	narrowRate = 32000
)

// The quantization classes, named for their steps, each with its steps, the bits of its code and
// whether it groups its samples.
var (
	// steps3 codes three samples of 3 steps in 5 bits.
	steps3 = quantClass{3, 5, true}
	// steps5 codes three samples of 5 steps in 7 bits.
	steps5 = quantClass{5, 7, true}
	// steps7 codes samples of 7 steps in 3 bits.
	steps7 = quantClass{7, 3, false}
	// steps9 codes three samples of 9 steps in 10 bits.
	steps9 = quantClass{9, 10, true}
	// steps15 codes samples of 15 steps in 4 bits.
	steps15 = quantClass{15, 4, false}
	// steps31 codes samples of 31 steps in 5 bits.
	steps31 = quantClass{31, 5, false}
	// steps63 codes samples of 63 steps in 6 bits.
	steps63 = quantClass{63, 6, false}
	// steps127 codes samples of 127 steps in 7 bits.
	steps127 = quantClass{127, 7, false}
	// steps255 codes samples of 255 steps in 8 bits.
	steps255 = quantClass{255, 8, false}
	// steps511 codes samples of 511 steps in 9 bits.
	steps511 = quantClass{511, 9, false}
	// steps1023 codes samples of 1023 steps in 10 bits.
	steps1023 = quantClass{1023, 10, false}
	// steps2047 codes samples of 2047 steps in 11 bits.
	steps2047 = quantClass{2047, 11, false}
	// steps4095 codes samples of 4095 steps in 12 bits.
	steps4095 = quantClass{4095, 12, false}
	// steps8191 codes samples of 8191 steps in 13 bits.
	steps8191 = quantClass{8191, 13, false}
	// steps16383 codes samples of 16383 steps in 14 bits.
	steps16383 = quantClass{16383, 14, false}
	// steps32767 codes samples of 32767 steps in 15 bits.
	steps32767 = quantClass{32767, 15, false}
	// steps65535 codes samples of 65535 steps in 16 bits.
	steps65535 = quantClass{65535, 16, false}
)

// The runs of subbands that the allocation tables share, each with its classes, its number of
// subbands and the width of its allocations.
var (
	// fineLow holds the lowest subbands of the tables of 27 and 30 subbands.
	fineLow = allocationRun{[]quantClass{
		steps3, steps7, steps15, steps31, steps63, steps127, steps255, steps511, steps1023, steps2047,
		steps4095, steps8191, steps16383, steps32767, steps65535,
	}, 3, 4}
	// fineMiddle holds the next subbands of the tables of 27 and 30 subbands.
	fineMiddle = allocationRun{[]quantClass{
		steps3, steps5, steps7, steps9, steps15, steps31, steps63, steps127, steps255, steps511,
		steps1023, steps2047, steps4095, steps8191, steps65535,
	}, 8, 4}
	// fineHigh holds the subbands above those of the tables of 27 and 30 subbands.
	fineHigh = allocationRun{[]quantClass{steps3, steps5, steps7, steps9, steps15, steps31, steps65535}, 12, 3}
	// fineTop holds the classes of the highest subbands of the tables of 27 and 30 subbands, 4 or 7
	// of them.
	fineTop = []quantClass{steps3, steps5, steps65535}
	// narrowLow holds the lowest subbands of the tables of 8 and 12 subbands.
	narrowLow = allocationRun{[]quantClass{
		steps3, steps5, steps9, steps15, steps31, steps63, steps127, steps255, steps511, steps1023,
		steps2047, steps4095, steps8191, steps16383, steps32767,
	}, 2, 4}
	// narrowHigh holds the classes of the higher subbands of the tables of 8 and 12 subbands, and
	// of the middle subbands of the lower-rate table.
	narrowHigh = []quantClass{steps3, steps5, steps9, steps15, steps31, steps63, steps127}
)

// The allocation tables, each a list of runs with their classes, numbers of subbands and widths.
var (
	// allocation27 is the MPEG-1 table of 27 subbands.
	allocation27 = allocationTable{fineLow, fineMiddle, fineHigh, {fineTop, 4, 2}}
	// allocation30 is the MPEG-1 table of 30 subbands.
	allocation30 = allocationTable{fineLow, fineMiddle, fineHigh, {fineTop, 7, 2}}
	// allocation8 is the MPEG-1 table of 8 subbands.
	allocation8 = allocationTable{narrowLow, {narrowHigh, 6, 3}}
	// allocation12 is the MPEG-1 table of 12 subbands, for 32 kHz.
	allocation12 = allocationTable{narrowLow, {narrowHigh, 10, 3}}
	// lowerRateAllocation is the table of 30 subbands for the lower sampling frequencies.
	lowerRateAllocation = allocationTable{
		{[]quantClass{
			steps3, steps5, steps7, steps9, steps15, steps31, steps63, steps127, steps255, steps511,
			steps1023, steps2047, steps4095, steps8191, steps16383,
		}, 4, 4},
		{narrowHigh, 7, 3},
		{[]quantClass{steps3, steps5, steps9}, 19, 2},
	}
)

// allocationFor returns the allocation table of a Layer II frame.
//
// Frames at the lower sampling frequencies share one table. An MPEG-1 frame's table follows its
// bitrate per channel. Below 56 kbit/s it has 8 subbands, or 12 at 32 kHz. Up to 80 kbit/s it
// has 27. Above that it has 27 at 48 kHz and 30 otherwise.
//
// Parameters:
//   - head: the frame's header, with the bitrate of a free-format stream.
//
// Returns:
//   - allocationTable: the table.
func allocationFor(head frame.Header) allocationTable {
	perChannel := head.Bitrate / head.Channels()

	switch {
	case head.Version != frame.MPEG1:
		return lowerRateAllocation
	case perChannel < narrowBelow && head.SampleRate == narrowRate:
		return allocation12
	case perChannel < narrowBelow:
		return allocation8
	case perChannel <= mediumUpTo || head.SampleRate == wideRate:
		return allocation27
	default:
		return allocation30
	}
}
