// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"encoding/binary"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// FuzzSideInfo parses side information from any bytes under any Layer III header and checks that
// every field lies within its width, that window switching leaves the regions it does not
// transmit at 0, and that granules and channels the frame lacks stay 0.
//
// The seed corpus holds the first frame of each encoded fixture.
func FuzzSideInfo(f *testing.F) {
	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "mpeg2.mp3", "mpeg25.mp3", "protected.mp3"} {
		data, err := os.ReadFile("../../testdata/" + name)
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

// FuzzSpectrum decodes spectral values from any bits under any side information and checks that
// whole pairs and quadruples come back, that every value past them is 0 and within the widest
// table's range, and that decoding stops with room for another quadruple only at or just past the
// end.
//
// It then requantizes them under scalefactors and gains drawn from the side information bytes, and
// checks that each spectral value is finite and shares its quantized value's sign and whether it is
// 0.
//
// A pair runs at most 47 bits past the end, a 19-bit code word with 13 linbits and a sign for each
// value. The seed corpus holds ones under long, short and mixed blocks, and no bits.
func FuzzSpectrum(f *testing.F) {
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}, []byte{0, 100, 1, 13, 24, 3, 2, 0, 0, 0, 0}, uint16(32))
	f.Add([]byte{0x93, 0x5a, 0x17, 0xc4, 0x00, 0x81}, []byte{1, 32, 7, 16, 0, 0, 0, 2, 1, 1, 4}, uint16(48))
	f.Add([]byte{0x00, 0x12, 0x7f}, []byte{1, 31, 2, 31, 0, 0, 0, 2, 0, 0, 8}, uint16(24))
	f.Add([]byte{}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, uint16(0))

	f.Fuzz(func(t *testing.T, data, fields []byte, end uint16) {
		if len(fields) < 11 {
			return
		}

		info := GranuleInfo{
			TableSelect:       [3]int{int(fields[2] % 32), int(fields[3] % 32), int(fields[4] % 32)},
			BigValues:         (int(fields[0])<<8 | int(fields[1])) % (maxBigValues + 1),
			Region0Count:      int(fields[5] % 16),
			Region1Count:      int(fields[6] % 8),
			Count1TableSelect: int(fields[8] % 2),
			BlockType:         blockTypes[fields[7]%4],
			MixedBlock:        fields[9]%2 == 1,
		}

		var (
			stream   bits.Reader
			spectrum Spectrum
		)

		stream.Reset(data, 0)
		spectrum.Read(&stream, frame.Header{SampleRate: bandTables[int(fields[10])%len(bandTables)].rate}, &info, int(end))

		require.Zero(t, spectrum.Decoded%2)
		require.LessOrEqual(t, spectrum.Decoded, granuleLines)

		for at, value := range spectrum.Values {
			if at >= spectrum.Decoded {
				require.Zero(t, value, "line %d", at)
			}

			require.LessOrEqual(t, max(value, -value), 15+1<<13-1, "line %d", at)
		}

		if spectrum.Decoded+4 <= granuleLines {
			assert.GreaterOrEqual(t, stream.Pos(), int(end))
		}

		assert.Less(t, stream.Pos(), int(end)+47)

		var (
			factors Scalefactors
			lines   [granuleLines]float64
		)

		for at := range factors.values {
			factors.values[at] = fields[at%len(fields)] % 16
		}

		info.GlobalGain = int(fields[0])
		info.ScalefacScale = fields[1]%2 == 1
		info.SubblockGain = [3]int{int(fields[2] % 8), int(fields[3] % 8), int(fields[4] % 8)}
		if info.BlockType == ShortBlocks && info.MixedBlock {
			factors.shift = mpeg1MixedShift
		}
		factors.preflag = fields[5]%2 == 1

		spectrum.Requantize(&lines, &info, &factors)

		for line, value := range lines {
			require.False(t, math.IsInf(value, 0) || math.IsNaN(value), "line %d", line)
			require.Equal(t, spectrum.Values[line] == 0, value == 0, "line %d", line)
			require.Equal(t, spectrum.Values[line] < 0, value < 0, "line %d", line)
		}
	})
}

// FuzzStereo restores joint stereo lines drawn from any bytes under any block, sampling frequency,
// mode, intensity scale and intensity positions, and checks every line against the formulas.
//
// It then reorders both channels and checks that each holds the same lines as before.
//
// Each pair of bytes gives a line's left and right values between -1 and 1, and lines past the
// bytes are 0. MPEG-1 positions run from 0 to 15, so most code none. At the lower sampling
// frequencies each position byte gives a width from 0 to 4 bits and a value up to the largest it
// holds. The seed corpus holds a long block with values in both channels, a short block with a
// silent right channel, mixed blocks with the right channel silent above line 40, and the same at
// the lower frequencies.
func FuzzStereo(f *testing.F) {
	f.Add(slices.Repeat([]byte{0x40, 0x20}, granuleLines), []byte{0, 0, 3, 3, 1, 4, 1, 5, 9, 2, 6})
	f.Add(slices.Repeat([]byte{0x7f, 0x00}, granuleLines), []byte{1, 3, 1, 0, 1, 2, 3, 4, 5, 6})
	f.Add(slices.Repeat([]byte{0x81, 0x33}, 20), []byte{2, 4, 3, 6, 6, 6, 0, 0, 0, 7, 8})
	f.Add(slices.Repeat([]byte{0x40, 0x20}, granuleLines), []byte{4, 0, 7, 14, 9, 19, 24, 4})
	f.Add(slices.Repeat([]byte{0x81, 0x33}, 20), []byte{8, 4, 1, 3, 8, 13, 18, 23})

	kinds := []GranuleInfo{
		{},
		{BlockType: StartBlock},
		{BlockType: StopBlock},
		{BlockType: ShortBlocks},
		{BlockType: ShortBlocks, MixedBlock: true},
	}

	f.Fuzz(func(t *testing.T, data, fields []byte) {
		if len(fields) < 4 {
			return
		}

		rate := int(fields[0]) % len(publishedWidths)
		widths := publishedWidths[rate]
		info := kinds[int(fields[1])%len(kinds)]
		info.ScalefacCompress = int(fields[2] >> 2 & 1)
		head := frame.Header{
			SampleRate: widths.rate, Version: frame.MPEG1, Layer: frame.Layer3,
			Mode: frame.JointStereo, Extension: fields[2] % 4,
		}

		if rate >= len(mpeg1Rates) {
			head.Version = frame.MPEG2
		}

		factors := Scalefactors{}

		if info.BlockType == ShortBlocks && info.MixedBlock {
			factors.shift = mpeg1MixedShift
			if head.Version != frame.MPEG1 {
				factors.shift = lsfMixedShift
			}
		}

		for at := range shortSent {
			field := fields[3+at%(len(fields)-3)]
			factors.values[at] = field % 16

			if head.Version != frame.MPEG1 {
				factors.widest[at] = 1<<(field%5) - 1
				factors.values[at] = field / 5 % (factors.widest[at] + 1)
			}
		}

		var left, right [granuleLines]float64

		for line := range min(len(data)/2, granuleLines) {
			left[line] = float64(int8(data[2*line])) / 128
			right[line] = float64(int8(data[2*line+1])) / 128
		}

		places := linePlaces(&info, widths.long, widths.short)
		wantLeft, wantRight := stereoOracle(head, &left, &right, &places, &info, &factors)

		Stereo(head, &left, &right, &info, &factors)

		assertLines(t, &wantLeft, &left, "left")
		assertLines(t, &wantRight, &right, "right")

		for _, lines := range []*[granuleLines]float64{&left, &right} {
			before := slices.Sorted(slices.Values(lines[:]))

			Reorder(head, lines, &info)
			require.Equal(t, before, slices.Sorted(slices.Values(lines[:])))
		}
	})
}
