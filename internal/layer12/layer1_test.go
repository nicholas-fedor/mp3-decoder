// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestReadAllocation checks that each allocation n gives samples of n+1 bits, that both channels
// share the allocations from the bound on, and that the forbidden allocation is reported once all
// the allocations are read.
func TestReadAllocation(t *testing.T) {
	t.Parallel()

	var fields mp3test.Layer1Audio

	fields.Allocation[0][0], fields.Allocation[1][0] = 1, 14
	fields.Allocation[0][3], fields.Allocation[1][3] = 0, 5
	fields.Allocation[0][4], fields.Allocation[1][4] = 7, 9

	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo}

	var decoder Decoder

	decoder.stream.Reset(head.Layer1Audio(&fields), 0)
	assert.True(t, decoder.readAllocation(2, 4))
	assert.Equal(t, [2]int{2, 15}, [2]int{decoder.widths[0][0], decoder.widths[1][0]})
	assert.Equal(t, [2]int{0, 6}, [2]int{decoder.widths[0][3], decoder.widths[1][3]})
	assert.Equal(t, [2]int{8, 8}, [2]int{decoder.widths[0][4], decoder.widths[1][4]})
	assert.Equal(t, 4*(2*4+28), decoder.stream.Pos())

	fields.Allocation[1][2] = 15

	decoder.stream.Reset(head.Layer1Audio(&fields), 0)
	assert.False(t, decoder.readAllocation(2, 4))
	assert.Equal(t, 4*(2*4+28), decoder.stream.Pos())
}

// TestReadSet checks the steps a scalefactor gives and the values sample codes stand for, below
// the bound and from it on.
//
// A code of 2 bits stands for -1, 0 or 1 steps of 2/3 times the scalefactor.
func TestReadSet(t *testing.T) {
	t.Parallel()

	var fields mp3test.Layer1Audio

	fields.Allocation[0][0], fields.Allocation[1][0] = 1, 1
	fields.Allocation[0][4] = 14
	fields.Scalefactors[0][0], fields.Scalefactors[1][0] = 0, 3
	fields.Scalefactors[0][4], fields.Scalefactors[1][4] = 6, 9
	fields.Samples[0][0][0], fields.Samples[0][1][0] = 0, 2
	fields.Samples[0][0][4] = 0x7ffe

	head := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Mode: mp3test.JointStereo}

	var decoder Decoder

	decoder.stream.Reset(head.Layer1Audio(&fields), 0)
	decoder.readAllocation(2, 4)
	decoder.readScalefactors(2)
	decoder.readSet(2, 4)

	assert.InDelta(t, -2*2.0/3, decoder.samples[0][0], 0)
	assert.InDelta(t, 1*2.0/3, decoder.samples[1][0], 0)
	assert.InDelta(t, 0.5*16383*2/32767.0, decoder.samples[0][4], 1e-16)
	assert.InDelta(t, 0.25*16383*2/32767.0, decoder.samples[1][4], 1e-16)
	assert.Zero(t, decoder.samples[0][1])
	assert.Zero(t, decoder.samples[1][31])
}
