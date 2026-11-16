// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrameIndex checks the entries of an empty index, of one with a step of one frame, and of
// one that filled and doubled its step.
func TestFrameIndex(t *testing.T) {
	t.Parallel()

	var index frameIndex

	_, _, ok := index.before(0)
	assert.False(t, ok, "an empty index has an entry")

	for frame := range int64(3 * indexSize) {
		index.add(frame, 10*frame)
		index.add(frame, -1) // A frame found again keeps its entry.
	}

	index.add(5*indexSize, -1) // A frame past the next one taken is not indexed.

	require.Len(t, index.offsets, 3*indexSize/4)
	assert.Equal(t, uint(2), index.shift)

	tests := []struct {
		frame, from int64
		ok          bool
	}{
		{-1, 0, false},
		{0, 0, true},
		{3, 0, true},
		{4, 4, true},
		{2001, 2000, true},
		{3*indexSize - 1, 3*indexSize - 4, true},
		{1 << 40, 3*indexSize - 4, true},
	}

	for _, tt := range tests {
		from, offset, ok := index.before(tt.frame)
		assert.Equal(t, tt.ok, ok, "frame %d", tt.frame)

		if tt.ok {
			assert.Equal(t, tt.from, from, "frame %d", tt.frame)
			assert.Equal(t, 10*tt.from, offset, "frame %d", tt.frame)
		}
	}
}
