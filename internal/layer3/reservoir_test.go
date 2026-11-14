// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReservoirSize checks that the reservoir holds 511 bytes, the largest main_data_begin of any
// version's layout.
func TestReservoirSize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 511, reservoirSize)

	for _, lay := range []layout{mpeg1Layout, lsfLayout} {
		assert.LessOrEqual(t, 1<<lay.begin-1, reservoirSize)
	}
}

// TestMainDataAllocs checks that handing out main data allocates nothing. It runs alone, since
// AllocsPerRun refuses a parallel test.
func TestMainDataAllocs(t *testing.T) {
	var reservoir Reservoir

	own := make([]byte, 400)

	allocs := testing.AllocsPerRun(100, func() {
		reservoir.MainData(300, own)
	})

	assert.Zero(t, allocs)
}
