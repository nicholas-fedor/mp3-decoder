// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"errors"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// BlockType is the kind of block a granule's channel holds, and the window it takes.
type BlockType uint8

// SideInfo is the side information of a Layer III frame.
type SideInfo struct {
	// Granules holds the side information of each granule of each channel, at [granule][channel].
	//
	// An MPEG-1 frame has two granules, and an MPEG-2 or MPEG-2.5 frame one.
	Granules [2][2]GranuleInfo
	// SCFSI says, for each channel and group of scalefactor bands, whether the second granule
	// reuses the first granule's scalefactors.
	//
	// Only MPEG-1 frames have it.
	SCFSI [2][4]bool
	// MainDataBegin is how many bytes before the frame's own main data its main data begins, in the
	// main data of the frames before it.
	MainDataBegin int
}

// GranuleInfo is the side information of one granule of one channel.
type GranuleInfo struct {
	// TableSelect holds the big_values table of each region.
	//
	// A granule with window switching has two regions, so its third entry is 0.
	TableSelect [3]int
	// SubblockGain holds the gain offset of each short window, which a granule with window
	// switching has.
	SubblockGain [3]int
	// Part23Length is the number of bits the granule's scalefactors and Huffman data take.
	Part23Length int
	// BigValues is the number of pairs in the big_values region, at most 288.
	BigValues int
	// GlobalGain is the quantizer step size.
	GlobalGain int
	// ScalefacCompress gives the number of bits each scalefactor takes.
	//
	// In MPEG-2 and MPEG-2.5 frames it also gives the preflag and, in the right channel of an
	// intensity stereo frame, the intensity scale.
	ScalefacCompress int
	// Region0Count is one less than the number of scalefactor bands in region0.
	//
	// Only a granule without window switching transmits it.
	Region0Count int
	// Region1Count is one less than the number of scalefactor bands in region1.
	//
	// Only a granule without window switching transmits it.
	Region1Count int
	// Count1TableSelect is the count1 table, 0 for table A and 1 for table B.
	Count1TableSelect int
	// BlockType is the kind of block the granule holds.
	BlockType BlockType
	// MixedBlock reports whether the lowest subbands hold a long block under short blocks.
	MixedBlock bool
	// Preflag reports whether the pretab amplifies the high scalefactor bands.
	//
	// Only MPEG-1 frames transmit it. MPEG-2 and MPEG-2.5 frames give it in ScalefacCompress.
	Preflag bool
	// ScalefacScale reports whether scalefactors step by 1 rather than by 0.5.
	ScalefacScale bool
}

// layout is the width of each side information field that differs between versions.
type layout struct {
	// begin is the width of main_data_begin.
	begin int
	// privateMono is the width of the private bits of a mono frame.
	privateMono int
	// privateStereo is the width of the private bits of a two-channel frame.
	privateStereo int
	// compress is the width of scalefac_compress.
	compress int
	// granules is the number of granules.
	granules int
	// scfsi reports whether the frame has scfsi bits.
	scfsi bool
	// preflag reports whether each granule has a preflag bit.
	preflag bool
}

// Block types.
const (
	// NormalBlock is one long block with the normal window.
	NormalBlock BlockType = iota
	// StartBlock is one long block with the start window, before short blocks.
	StartBlock
	// ShortBlocks is three short blocks with short windows.
	ShortBlocks
	// StopBlock is one long block with the stop window, after short blocks.
	StopBlock
)

// bitsPerByte is the number of bits in a byte.
const bitsPerByte = 8

// Side information field widths shared by every version.
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
	// maxBigValues is the most pairs a granule's 576 values hold.
	maxBigValues = 288
	// switchedRegions is the number of regions of a granule with window switching.
	switchedRegions = 2
)

// Widths of the MPEG-1 side information fields that differ between versions.
const (
	// mpeg1Begin is the width of main_data_begin.
	mpeg1Begin = 9
	// mpeg1PrivateMono is the width of the private bits of a mono frame.
	mpeg1PrivateMono = 5
	// mpeg1PrivateStereo is the width of the private bits of a two-channel frame.
	mpeg1PrivateStereo = 3
	// mpeg1Compress is the width of scalefac_compress.
	mpeg1Compress = 4
	// mpeg1Granules is the number of granules.
	mpeg1Granules = 2
)

// Widths of the MPEG-2 and MPEG-2.5 side information fields that differ between versions.
const (
	// lsfBegin is the width of main_data_begin.
	lsfBegin = 8
	// lsfPrivateMono is the width of the private bits of a mono frame.
	lsfPrivateMono = 1
	// lsfPrivateStereo is the width of the private bits of a two-channel frame.
	lsfPrivateStereo = 2
	// lsfCompress is the width of scalefac_compress.
	lsfCompress = 9
	// lsfGranules is the number of granules.
	lsfGranules = 1
)

// Side information layouts.
var (
	// mpeg1Layout is the layout of MPEG-1 frames.
	mpeg1Layout = layout{
		begin: mpeg1Begin, privateMono: mpeg1PrivateMono, privateStereo: mpeg1PrivateStereo,
		compress: mpeg1Compress, granules: mpeg1Granules, scfsi: true, preflag: true,
	}
	// lsfLayout is the layout of MPEG-2 and MPEG-2.5 frames.
	lsfLayout = layout{
		begin: lsfBegin, privateMono: lsfPrivateMono, privateStereo: lsfPrivateStereo,
		compress: lsfCompress, granules: lsfGranules, scfsi: false, preflag: false,
	}
)

// blockTypes maps the two block_type bits to a block type.
var blockTypes = [4]BlockType{NormalBlock, StartBlock, ShortBlocks, StopBlock}

// ErrBlockType is returned for a granule whose window switching flag is set with block type 0,
// which no granule may have.
var ErrBlockType = errors.New("layer3: window switching with block type 0")

// ErrShortFrame is returned for a frame that ends before its side information does.
var ErrShortFrame = errors.New("layer3: frame ends before its side information")

// Parse reads the side information of a Layer III frame.
//
// A big_values count above 288 is read as 288, the most pairs 576 values hold.
//
// Parameters:
//   - head: the frame's header.
//   - data: the frame's bytes from its header onward.
//
// Returns:
//   - error: [ErrShortFrame] or [ErrBlockType].
func (s *SideInfo) Parse(head frame.Header, data []byte) error {
	var zero SideInfo

	*s = zero

	if len(data) < head.DataAt() {
		return ErrShortFrame
	}

	lay := lsfLayout
	if head.Version == frame.MPEG1 {
		lay = mpeg1Layout
	}

	var stream bits.Reader

	stream.Reset(data, (head.DataAt()-head.SideInfoSize())*bitsPerByte)

	s.MainDataBegin = int(stream.Read(lay.begin))

	if head.Channels() == 1 {
		stream.Skip(lay.privateMono)
	} else {
		stream.Skip(lay.privateStereo)
	}

	if lay.scfsi {
		for ch := range head.Channels() {
			for group := range s.SCFSI[ch] {
				s.SCFSI[ch][group] = stream.Read(1) == 1
			}
		}
	}

	for gr := range lay.granules {
		for ch := range head.Channels() {
			err := s.Granules[gr][ch].parse(&stream, lay)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// parse reads the side information of one granule of one channel.
//
// Parameters:
//   - stream: the bitstream, at the granule's side information.
//   - lay: the layout of the frame's version.
//
// Returns:
//   - error: [ErrBlockType].
func (g *GranuleInfo) parse(stream *bits.Reader, lay layout) error {
	g.Part23Length = int(stream.Read(part23Bits))
	g.BigValues = min(int(stream.Read(bigValuesBits)), maxBigValues)
	g.GlobalGain = int(stream.Read(globalGainBits))
	g.ScalefacCompress = int(stream.Read(lay.compress))

	if stream.Read(1) == 1 {
		g.BlockType = blockTypes[stream.Read(blockTypeBits)]
		if g.BlockType == NormalBlock {
			return ErrBlockType
		}

		g.MixedBlock = stream.Read(1) == 1

		for region := range switchedRegions {
			g.TableSelect[region] = int(stream.Read(tableSelectBits))
		}

		for window := range g.SubblockGain {
			g.SubblockGain[window] = int(stream.Read(subblockGainBits))
		}
	} else {
		for region := range g.TableSelect {
			g.TableSelect[region] = int(stream.Read(tableSelectBits))
		}

		g.Region0Count = int(stream.Read(region0Bits))
		g.Region1Count = int(stream.Read(region1Bits))
	}

	if lay.preflag {
		g.Preflag = stream.Read(1) == 1
	}

	g.ScalefacScale = stream.Read(1) == 1
	g.Count1TableSelect = int(stream.Read(1))

	return nil
}
