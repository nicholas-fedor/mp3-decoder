// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// TestQuantClasses checks that a class's samples fill the codes of its width. A class coded alone
// has one step fewer than its codes, and a grouped class needs every bit of its code word for
// three samples.
func TestQuantClasses(t *testing.T) {
	t.Parallel()

	classes := []quantClass{
		steps3, steps5, steps7, steps9, steps15, steps31, steps63, steps127, steps255, steps511, steps1023,
		steps2047, steps4095, steps8191, steps16383, steps32767, steps65535,
	}

	for _, class := range classes {
		if !class.grouped {
			assert.Equal(t, 1<<class.bits-1, class.steps, "%d steps", class.steps)

			continue
		}

		cubed := class.steps * class.steps * class.steps

		assert.Greater(t, cubed, 1<<(class.bits-1), "%d steps", class.steps)
		assert.LessOrEqual(t, cubed, 1<<class.bits, "%d steps", class.steps)
	}

	assert.Equal(t, []int{3, 5, 9}, []int{steps3.steps, steps5.steps, steps9.steps})
	assert.True(t, steps3.grouped && steps5.grouped && steps9.grouped)
}

// TestAllocationTables checks each table's subbands, that each run selects a class for every
// allocation its width holds in rising steps, and that the tables of 30 and 12 subbands extend
// their runs of 27 and 8.
func TestAllocationTables(t *testing.T) {
	t.Parallel()

	tables := map[string]struct {
		table    allocationTable
		subbands int
	}{
		"27": {allocation27, 27}, "30": {allocation30, 30}, "8": {allocation8, 8},
		"12": {allocation12, 12}, "lower rates": {lowerRateAllocation, 30},
	}

	for name, tt := range tables {
		assert.Equal(t, tt.subbands, subbandsOf(tt.table), name)

		for at, run := range tt.table {
			assert.Len(t, run.classes, 1<<run.width-1, "%s run %d", name, at)

			for class := 1; class < len(run.classes); class++ {
				assert.Greater(t, run.classes[class].steps, run.classes[class-1].steps, "%s run %d", name, at)
			}
		}
	}

	assert.Equal(t, allocation27[:3], allocation30[:3])
	assert.Equal(t, allocation27[3].classes, allocation30[3].classes)
	assert.Equal(t, allocation8[0], allocation12[0])
	assert.Equal(t, allocation8[1].classes, allocation12[1].classes)
}

// TestAllocationFor checks the table of every fixed MPEG-1 bitrate, sampling frequency and number
// of channels against the choice mpg123 lists by bitrate index, then the lower sampling
// frequencies and free-format bitrates between those the lists name.
func TestAllocationFor(t *testing.T) {
	t.Parallel()

	bitrates := []int{32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384}
	rates := []int{44100, 48000, 32000}

	// mpg123's choice by sampling frequency, number of channels and bitrate index from 1, as the
	// number of subbands of the table.
	chosen := [3][2][14]int{
		{
			{8, 8, 8, 8, 8, 8, 27, 27, 27, 30, 30, 30, 30, 30},
			{8, 8, 27, 27, 27, 30, 30, 30, 30, 30, 30, 30, 30, 30},
		},
		{
			{8, 8, 8, 8, 8, 8, 27, 27, 27, 27, 27, 27, 27, 27},
			{8, 8, 27, 27, 27, 27, 27, 27, 27, 27, 27, 27, 27, 27},
		},
		{
			{12, 12, 12, 12, 12, 12, 27, 27, 27, 30, 30, 30, 30, 30},
			{12, 12, 27, 27, 27, 30, 30, 30, 30, 30, 30, 30, 30, 30},
		},
	}

	for at, rate := range rates {
		for mono, mode := range []frame.Mode{frame.Stereo, frame.Mono} {
			for index, bitrate := range bitrates {
				head := frame.Header{SampleRate: rate, Bitrate: 1000 * bitrate, Version: frame.MPEG1, Layer: frame.Layer2, Mode: mode}

				assert.Equal(t, chosen[at][mono][index], subbandsOf(allocationFor(head)), "%d Hz, %v, %d kbit/s", rate, mode, bitrate)
			}
		}
	}

	lower := frame.Header{SampleRate: 24000, Bitrate: 160000, Version: frame.MPEG2, Layer: frame.Layer2, Mode: frame.Stereo}
	assert.Equal(t, lowerRateAllocation, allocationFor(lower))

	free := []struct {
		rate, perChannel, want int
	}{
		{44100, 50000, 8}, {32000, 50000, 12}, {44100, 90000, 30}, {48000, 90000, 27}, {44100, 500000, 30},
	}

	for _, tt := range free {
		head := frame.Header{SampleRate: tt.rate, Bitrate: tt.perChannel, Version: frame.MPEG1, Layer: frame.Layer2, Mode: frame.Mono, Free: true}
		assert.Equal(t, tt.want, subbandsOf(allocationFor(head)), "free format at %d Hz and %d bit/s", tt.rate, tt.perChannel)
	}
}

// subbandsOf returns the number of subbands an allocation table holds.
func subbandsOf(table allocationTable) int {
	subbands := 0

	for _, run := range table {
		subbands += run.subbands
	}

	return subbands
}
