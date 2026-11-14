// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"encoding/binary"
	"os"
	"slices"
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

// FuzzReservoir feeds a reservoir frames that any bytes script and checks each frame's main data
// and missing count against a history of every byte added since the last reset.
//
// Each frame takes three bytes. A first byte of 0 resets the reservoir instead. The first two give
// main_data_begin, up to 599 to cover a begin past the 511 bytes the reservoir holds, and the third
// gives the frame's size in steps of 16 bytes, up to 4080 to cover frames past the largest. The seed
// corpus holds a stream start, a reach across frames, an underflow after a reset and a frame past
// the largest.
func FuzzReservoir(f *testing.F) {
	f.Add([]byte{0x01, 0x00, 0x10})
	f.Add([]byte{0x01, 0x00, 0x04, 0x01, 0x00, 0x04, 0x01, 0xff, 0x01})
	f.Add([]byte{0x01, 0x00, 0x10, 0x00, 0x01, 0x40, 0x02})
	f.Add([]byte{0x01, 0x00, 0xff, 0x01, 0xff, 0x01})

	f.Fuzz(func(t *testing.T, script []byte) {
		var (
			reservoir Reservoir
			history   []byte
		)

		for len(script) > 0 {
			if script[0] == 0 {
				reservoir.Reset()

				history, script = history[:0], script[1:]

				continue
			}

			if len(script) < 3 {
				return
			}

			begin := int(binary.BigEndian.Uint16(script) % 600)
			own := make([]byte, int(script[2])*16)

			for i := range own {
				own[i] = byte(len(history) + i)
			}

			script = script[3:]

			main, missing := reservoir.MainData(begin, own)

			kept := own[:min(len(own), frame.MaxSize)]
			reach := min(begin, len(history), reservoirSize)

			assert.Equal(t, string(slices.Concat(history[len(history)-reach:], kept)), string(main))
			assert.Equal(t, begin-reach, missing)

			history = append(history, kept...)
		}
	})
}
