// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3_test

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// frameWith builds a frame of head holding side information after its header and any CRC.
func frameWith(t *testing.T, head mp3test.Header, side *mp3test.SideInfo) ([]byte, frame.Header) {
	t.Helper()

	parsed, ok := frame.Parse(head.Bytes())
	require.True(t, ok)

	data := head.Frame(parsed.DataAt() + 64)
	mp3test.Place(data, parsed.DataAt()-parsed.SideInfoSize(), head.SideInfo(side))

	return data, parsed
}

// want returns the side information a granule's raw fields read as.
func want(g mp3test.Granule, mpeg1 bool) layer3.GranuleInfo {
	info := layer3.GranuleInfo{
		Part23Length: int(g.Part23Length), BigValues: min(int(g.BigValues), 288), GlobalGain: int(g.GlobalGain),
		ScalefacCompress: int(g.ScalefacCompress), ScalefacScale: g.ScalefacScale,
		Count1TableSelect: int(g.Count1TableSelect), Preflag: mpeg1 && g.Preflag,
	}

	if g.WindowSwitching {
		info.BlockType = layer3.BlockType(g.BlockType)
		info.MixedBlock = g.MixedBlock
		info.TableSelect = [3]int{int(g.TableSelect[0]), int(g.TableSelect[1]), 0}

		for window, gain := range g.SubblockGain {
			info.SubblockGain[window] = int(gain)
		}

		return info
	}

	for region, table := range g.TableSelect {
		info.TableSelect[region] = int(table)
	}

	info.Region0Count, info.Region1Count = int(g.Region0Count), int(g.Region1Count)

	return info
}

// full is a granule with every field at its widest.
var full = mp3test.Granule{
	Part23Length: 4095, BigValues: 288, GlobalGain: 255, ScalefacCompress: 15,
	TableSelect: [3]uint32{31, 30, 29}, Region0Count: 15, Region1Count: 7, Count1TableSelect: 1,
	Preflag: true, ScalefacScale: true,
}

// TestParseLayouts reads back side information in each version's layout, mono and stereo, with
// each switched block type, a CRC, and fields at their widest.
func TestParseLayouts(t *testing.T) {
	t.Parallel()

	short := mp3test.Granule{
		Part23Length: 1000, BigValues: 100, GlobalGain: 140, ScalefacCompress: 9,
		WindowSwitching: true, BlockType: 2, MixedBlock: true, TableSelect: [3]uint32{7, 13, 0},
		SubblockGain: [3]uint32{7, 0, 5},
	}
	start := short
	start.BlockType, start.MixedBlock = 1, false
	stop := short
	stop.BlockType = 3

	lsf := full
	lsf.ScalefacCompress = 511

	tests := []struct {
		name  string
		head  mp3test.Header
		side  mp3test.SideInfo
		mpeg1 bool
	}{
		{
			name: "MPEG-1 joint stereo", mpeg1: true,
			head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9, Mode: mp3test.JointStereo},
			side: mp3test.SideInfo{
				MainDataBegin: 511, PrivateBits: 5, SCFSI: [2][4]bool{{true, false, true, false}, {false, true, false, true}},
				Granules: [2][2]mp3test.Granule{{full, short}, {start, stop}},
			},
		},
		{
			name: "MPEG-1 mono with a CRC", mpeg1: true,
			head: mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9, Mode: mp3test.Mono, Protected: true},
			side: mp3test.SideInfo{
				MainDataBegin: 300, SCFSI: [2][4]bool{{true, true, true, true}},
				Granules: [2][2]mp3test.Granule{{short}, {full}},
			},
		},
		{
			name: "MPEG-2 stereo",
			head: mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8, Mode: mp3test.Stereo},
			side: mp3test.SideInfo{MainDataBegin: 255, PrivateBits: 3, Granules: [2][2]mp3test.Granule{{lsf, stop}}},
		},
		{
			name: "MPEG-2.5 mono",
			head: mp3test.Header{Version: mp3test.MPEG25, Layer: mp3test.Layer3, Bitrate: 2, Mode: mp3test.Mono},
			side: mp3test.SideInfo{MainDataBegin: 17, Granules: [2][2]mp3test.Granule{{short}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, head := frameWith(t, tt.head, &tt.side)

			var got layer3.SideInfo

			require.NoError(t, got.Parse(head, data))
			assert.Equal(t, int(tt.side.MainDataBegin), got.MainDataBegin)

			granules := 1
			if tt.mpeg1 {
				granules = 2

				assert.Equal(t, tt.side.SCFSI[:head.Channels()], got.SCFSI[:head.Channels()])
			}

			for gr := range granules {
				for ch := range head.Channels() {
					assert.Equal(t, want(tt.side.Granules[gr][ch], tt.mpeg1), got.Granules[gr][ch], "granule %d channel %d", gr, ch)
				}
			}
		})
	}
}

// TestParseDecisions covers the readings that follow the leading decoders.
//
// The cases are big_values above 288, window switching with block type 0, a frame cut short, and a
// value reused across frames.
func TestParseDecisions(t *testing.T) {
	t.Parallel()

	mono := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9, Mode: mp3test.Mono}

	over := mp3test.SideInfo{Granules: [2][2]mp3test.Granule{{{BigValues: 300}}}}
	data, head := frameWith(t, mono, &over)

	var got layer3.SideInfo

	require.NoError(t, got.Parse(head, data))
	assert.Equal(t, 288, got.Granules[0][0].BigValues)

	zero := mp3test.SideInfo{Granules: [2][2]mp3test.Granule{{}, {{WindowSwitching: true, BlockType: 0}}}}
	data, head = frameWith(t, mono, &zero)
	require.ErrorIs(t, got.Parse(head, data), layer3.ErrBlockType)

	require.ErrorIs(t, got.Parse(head, data[:head.DataAt()-1]), layer3.ErrShortFrame)

	reused := mp3test.SideInfo{Granules: [2][2]mp3test.Granule{{full}, {full}}}
	data, head = frameWith(t, mono, &reused)
	require.NoError(t, got.Parse(head, data))

	lsfMono := mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8, Mode: mp3test.Mono}
	data, head = frameWith(t, lsfMono, &mp3test.SideInfo{})
	require.NoError(t, got.Parse(head, data))
	assert.Equal(t, layer3.SideInfo{}, got)
}

// TestParseEncodedFiles reads the side information of every frame of files LAME encoded and checks
// that each frame's granules fit between main_data_begin and the frame's end.
//
// The files hold clicks whose onsets make LAME switch to short blocks, and each must use them.
func TestParseEncodedFiles(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "mpeg2.mp3", "mpeg25.mp3", "protected.mp3"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile("testdata/" + name)
			require.NoError(t, err)

			at, first, ok := frame.Find(data, nil)
			require.True(t, ok)

			var side layer3.SideInfo

			blocks := map[layer3.BlockType]int{}

			for at < len(data) {
				head, ok := frame.Parse(data[at:])
				require.True(t, ok)
				require.True(t, first.SameStream(head))

				end := at + head.Size()
				require.NoError(t, side.Parse(head, data[at:end]))

				bits := 0

				for _, granule := range side.Granules {
					for _, info := range granule[:head.Channels()] {
						bits += info.Part23Length
						blocks[info.BlockType]++
					}
				}

				assert.LessOrEqual(t, bits, 8*(side.MainDataBegin+head.Size()-head.DataAt()), "frame at %d", at)

				at = end
			}

			assert.Positive(t, blocks[layer3.ShortBlocks], "short blocks")
			assert.Positive(t, slices.Max([]int{blocks[layer3.StartBlock], blocks[layer3.StopBlock]}), "start or stop blocks")
		})
	}
}
