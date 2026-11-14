// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// FuzzSideInfo parses side information from any bytes under any Layer III header and checks that
// every field lies within its width, that window switching leaves the regions it does not
// transmit at 0, and that granules and channels the frame lacks stay 0.
//
// The seed corpus holds the first frame of each encoded fixture.
func FuzzSideInfo(f *testing.F) {
	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "mpeg2.mp3", "mpeg25.mp3", "protected.mp3"} {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(f, err)

		head, ok := frame.Parse(data)
		require.True(f, ok)

		f.Add(data[:head.Size()])
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		head, ok := frame.Parse(data)
		if !ok || head.Layer != frame.Layer3 {
			return
		}

		var side SideInfo

		if side.Parse(head, data) != nil {
			return
		}

		lay := lsfLayout
		if head.Version == frame.MPEG1 {
			lay = mpeg1Layout
		}

		assert.Less(t, side.MainDataBegin, 1<<lay.begin)

		for gr := range side.Granules {
			for ch := range side.Granules[gr] {
				info := side.Granules[gr][ch]

				if gr >= lay.granules || ch >= head.Channels() {
					assert.Equal(t, GranuleInfo{}, info)

					continue
				}

				checkGranule(t, info, lay)
			}
		}
	})
}

// checkGranule checks one granule's fields against their widths and window switching.
func checkGranule(t *testing.T, info GranuleInfo, lay layout) {
	t.Helper()

	assert.Less(t, info.Part23Length, 1<<part23Bits)
	assert.LessOrEqual(t, info.BigValues, maxBigValues)
	assert.Less(t, info.GlobalGain, 1<<globalGainBits)
	assert.Less(t, info.ScalefacCompress, 1<<lay.compress)
	assert.Less(t, info.Count1TableSelect, 2)
	assert.False(t, info.Preflag && !lay.preflag)

	for _, table := range info.TableSelect {
		assert.Less(t, table, 1<<tableSelectBits)
	}

	if info.BlockType == NormalBlock {
		assert.False(t, info.MixedBlock)
		assert.Equal(t, [3]int{}, info.SubblockGain)

		return
	}

	assert.Zero(t, info.TableSelect[2])
	assert.Zero(t, info.Region0Count)
	assert.Zero(t, info.Region1Count)
}
