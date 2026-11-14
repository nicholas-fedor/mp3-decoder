// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

// Granule holds the raw side information fields of one granule of one channel.
type Granule struct {
	// TableSelect holds each region's table_select, of which window switching writes two.
	TableSelect [3]uint32
	// SubblockGain holds each short window's subblock_gain, which window switching writes.
	SubblockGain [3]uint32
	// Part23Length is part2_3_length.
	Part23Length uint32
	// BigValues is big_values.
	BigValues uint32
	// GlobalGain is global_gain.
	GlobalGain uint32
	// ScalefacCompress is scalefac_compress.
	ScalefacCompress uint32
	// BlockType is block_type, which window switching writes.
	BlockType uint32
	// Region0Count is region0_count, which a granule without window switching writes.
	Region0Count uint32
	// Region1Count is region1_count, which a granule without window switching writes.
	Region1Count uint32
	// Count1TableSelect is count1table_select.
	Count1TableSelect uint32
	// WindowSwitching is window_switching_flag.
	WindowSwitching bool
	// MixedBlock is mixed_block_flag, which window switching writes.
	MixedBlock bool
	// Preflag is preflag, which only MPEG-1 writes.
	Preflag bool
	// ScalefacScale is scalefac_scale.
	ScalefacScale bool
}

// SideInfo holds the raw side information fields of a Layer III frame.
type SideInfo struct {
	// Granules holds each granule's fields for each channel, at [granule][channel].
	Granules [2][2]Granule
	// SCFSI holds each channel's scfsi bits, which only MPEG-1 writes.
	SCFSI [2][4]bool
	// MainDataBegin is main_data_begin.
	MainDataBegin uint32
	// PrivateBits is private_bits.
	PrivateBits uint32
}

// sideLayout is the width of each side information field that differs between versions.
type sideLayout struct {
	// begin is the width of main_data_begin.
	begin int
	// privateMono is the width of a mono frame's private bits.
	privateMono int
	// privateStereo is the width of a two-channel frame's private bits.
	privateStereo int
	// compress is the width of scalefac_compress.
	compress int
	// granules is the number of granules.
	granules int
	// scfsi reports whether the frame writes scfsi bits.
	scfsi bool
	// preflag reports whether each granule writes preflag.
	preflag bool
}

// Side information widths of every version.
const (
	// part23Bits is the width of part2_3_length.
	part23Bits = 12
	// bigValuesBits is the width of big_values.
	bigValuesBits = 9
	// globalGainBits is the width of global_gain.
	globalGainBits = 8
	// blockTypeBits is the width of block_type.
	blockTypeBits = 2
	// tableSelectBits is the width of table_select.
	tableSelectBits = 5
	// subblockGainBits is the width of subblock_gain.
	subblockGainBits = 3
	// region0Bits is the width of region0_count.
	region0Bits = 4
	// region1Bits is the width of region1_count.
	region1Bits = 3
	// switchedRegions is the number of table_select fields window switching writes.
	switchedRegions = 2
)

// Side information widths of MPEG-1 frames.
const (
	// mpeg1BeginBits is the width of main_data_begin.
	mpeg1BeginBits = 9
	// mpeg1PrivateMono is the width of a mono frame's private bits.
	mpeg1PrivateMono = 5
	// mpeg1PrivateStereo is the width of a two-channel frame's private bits.
	mpeg1PrivateStereo = 3
	// mpeg1CompressBits is the width of scalefac_compress.
	mpeg1CompressBits = 4
	// mpeg1Granules is the number of granules.
	mpeg1Granules = 2
)

// Side information widths of MPEG-2 and MPEG-2.5 frames.
const (
	// lsfBeginBits is the width of main_data_begin.
	lsfBeginBits = 8
	// lsfPrivateMono is the width of a mono frame's private bits.
	lsfPrivateMono = 1
	// lsfPrivateStereo is the width of a two-channel frame's private bits.
	lsfPrivateStereo = 2
	// lsfCompressBits is the width of scalefac_compress.
	lsfCompressBits = 9
	// lsfGranules is the number of granules.
	lsfGranules = 1
)

// Side information layouts.
var (
	// mpeg1Side is the layout of MPEG-1 frames.
	mpeg1Side = sideLayout{
		begin: mpeg1BeginBits, privateMono: mpeg1PrivateMono, privateStereo: mpeg1PrivateStereo,
		compress: mpeg1CompressBits, granules: mpeg1Granules, scfsi: true, preflag: true,
	}
	// lsfSide is the layout of MPEG-2 and MPEG-2.5 frames.
	lsfSide = sideLayout{
		begin: lsfBeginBits, privateMono: lsfPrivateMono, privateStereo: lsfPrivateStereo,
		compress: lsfCompressBits, granules: lsfGranules, scfsi: false, preflag: false,
	}
)

// bit maps a boolean field to its bit.
var bit = map[bool]uint32{false: 0, true: 1}

// SideInfo encodes side information in the layout of the header's version and mode.
//
// Parameters:
//   - fields: the fields.
//
// Returns:
//   - []byte: the side information, which fills whole bytes.
func (h Header) SideInfo(fields *SideInfo) []byte {
	var writer BitWriter

	lay := lsfSide
	if h.Version == MPEG1 {
		lay = mpeg1Side
	}

	channels, private := 2, lay.privateStereo
	if h.Mode == Mono {
		channels, private = 1, lay.privateMono
	}

	writer.Write(fields.MainDataBegin, lay.begin)
	writer.Write(fields.PrivateBits, private)

	if lay.scfsi {
		for ch := range channels {
			for _, set := range fields.SCFSI[ch] {
				writer.Write(bit[set], 1)
			}
		}
	}

	for gr := range lay.granules {
		for ch := range channels {
			writer.granule(&fields.Granules[gr][ch], lay)
		}
	}

	return writer.Bytes()
}

// granule writes one granule's side information for one channel.
//
// Parameters:
//   - granule: the fields.
//   - lay: the layout of the frame's version.
func (w *BitWriter) granule(granule *Granule, lay sideLayout) {
	w.Write(granule.Part23Length, part23Bits)
	w.Write(granule.BigValues, bigValuesBits)
	w.Write(granule.GlobalGain, globalGainBits)
	w.Write(granule.ScalefacCompress, lay.compress)
	w.Write(bit[granule.WindowSwitching], 1)

	if granule.WindowSwitching {
		w.Write(granule.BlockType, blockTypeBits)
		w.Write(bit[granule.MixedBlock], 1)

		for _, table := range granule.TableSelect[:switchedRegions] {
			w.Write(table, tableSelectBits)
		}

		for _, gain := range granule.SubblockGain {
			w.Write(gain, subblockGainBits)
		}
	} else {
		for _, table := range granule.TableSelect {
			w.Write(table, tableSelectBits)
		}

		w.Write(granule.Region0Count, region0Bits)
		w.Write(granule.Region1Count, region1Bits)
	}

	if lay.preflag {
		w.Write(bit[granule.Preflag], 1)
	}

	w.Write(bit[granule.ScalefacScale], 1)
	w.Write(granule.Count1TableSelect, 1)
}
