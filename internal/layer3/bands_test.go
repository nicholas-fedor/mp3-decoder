// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBandTables checks the bands of each sampling frequency against a second transcription that
// gives each band's width, and checks that long bands cover 576 lines and short bands 192.
func TestBandTables(t *testing.T) {
	t.Parallel()

	widths := []struct {
		long  [longBands]int
		short [shortBands]int
		rate  int
	}{
		{
			[longBands]int{4, 4, 4, 4, 4, 4, 6, 6, 8, 8, 10, 12, 16, 20, 24, 28, 34, 42, 50, 54, 76, 158},
			[shortBands]int{4, 4, 4, 4, 6, 8, 10, 12, 14, 18, 22, 30, 56},
			44100,
		},
		{
			[longBands]int{4, 4, 4, 4, 4, 4, 6, 6, 6, 8, 10, 12, 16, 18, 22, 28, 34, 40, 46, 54, 54, 192},
			[shortBands]int{4, 4, 4, 4, 6, 6, 10, 12, 14, 16, 20, 26, 66},
			48000,
		},
		{
			[longBands]int{4, 4, 4, 4, 4, 4, 6, 6, 8, 10, 12, 16, 20, 24, 30, 38, 46, 56, 68, 84, 102, 26},
			[shortBands]int{4, 4, 4, 4, 6, 8, 12, 16, 20, 26, 34, 42, 12},
			32000,
		},
		{
			[longBands]int{6, 6, 6, 6, 6, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 38, 46, 52, 60, 68, 58, 54},
			[shortBands]int{4, 4, 4, 6, 6, 8, 10, 14, 18, 26, 32, 42, 18},
			22050,
		},
		{
			[longBands]int{6, 6, 6, 6, 6, 6, 8, 10, 12, 14, 16, 18, 22, 26, 32, 38, 46, 54, 62, 70, 76, 36},
			[shortBands]int{4, 4, 4, 6, 8, 10, 12, 14, 18, 24, 32, 44, 12},
			24000,
		},
		{
			[longBands]int{6, 6, 6, 6, 6, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 38, 46, 52, 60, 68, 58, 54},
			[shortBands]int{4, 4, 4, 6, 8, 10, 12, 14, 18, 24, 30, 40, 18},
			16000,
		},
		{
			[longBands]int{6, 6, 6, 6, 6, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 38, 46, 52, 60, 68, 58, 54},
			[shortBands]int{4, 4, 4, 6, 8, 10, 12, 14, 18, 24, 30, 40, 18},
			11025,
		},
		{
			[longBands]int{6, 6, 6, 6, 6, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 38, 46, 52, 60, 68, 58, 54},
			[shortBands]int{4, 4, 4, 6, 8, 10, 12, 14, 18, 24, 30, 40, 18},
			12000,
		},
		{
			[longBands]int{12, 12, 12, 12, 12, 12, 16, 20, 24, 28, 32, 40, 48, 56, 64, 76, 90, 2, 2, 2, 2, 2},
			[shortBands]int{8, 8, 8, 12, 16, 20, 24, 28, 36, 2, 2, 2, 26},
			8000,
		},
	}

	require.Len(t, bandTables, len(widths))

	for _, want := range widths {
		got := bandsAt(want.rate)
		require.NotNil(t, got, "%d Hz", want.rate)

		for band, width := range want.long {
			assert.Equal(t, width, got.long[band+1]-got.long[band], "%d Hz long band %d", want.rate, band)
		}

		for band, width := range want.short {
			assert.Equal(t, width, got.short[band+1]-got.short[band], "%d Hz short band %d", want.rate, band)
		}

		assert.Zero(t, got.long[0])
		assert.Equal(t, granuleLines, got.long[longBands], "%d Hz", want.rate)
		assert.Zero(t, got.short[0])
		assert.Equal(t, granuleLines/windows, got.short[shortBands], "%d Hz", want.rate)
	}

	assert.Nil(t, bandsAt(44000))
}
