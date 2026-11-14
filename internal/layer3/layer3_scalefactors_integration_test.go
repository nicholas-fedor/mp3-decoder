// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// block is a kind of block, as the builder writes it and as the side information reads.
type block struct {
	// name names the kind.
	name string
	// raw is the kind's raw side information fields.
	raw mp3test.Granule
	// info is the kind's side information.
	info layer3.GranuleInfo
}

// blocks lists every kind of block.
var blocks = []block{
	{"long", mp3test.Granule{}, layer3.GranuleInfo{}},
	{"start", mp3test.Granule{WindowSwitching: true, BlockType: 1}, layer3.GranuleInfo{BlockType: layer3.StartBlock}},
	{"stop", mp3test.Granule{WindowSwitching: true, BlockType: 3}, layer3.GranuleInfo{BlockType: layer3.StopBlock}},
	{"short", mp3test.Granule{WindowSwitching: true, BlockType: 2}, layer3.GranuleInfo{BlockType: layer3.ShortBlocks}},
	{
		"mixed",
		mp3test.Granule{WindowSwitching: true, BlockType: 2, MixedBlock: true},
		layer3.GranuleInfo{BlockType: layer3.ShortBlocks, MixedBlock: true},
	},
}

// slens holds the published slen1 and slen2 at each scalefac_compress.
var slens = [16][2]int{
	{0, 0},
	{0, 1},
	{0, 2},
	{0, 3},
	{3, 0},
	{1, 1},
	{1, 2},
	{1, 3},
	{2, 1},
	{2, 2},
	{2, 3},
	{3, 1},
	{3, 2},
	{3, 3},
	{4, 2},
	{4, 3},
}

// TestReadMPEG1RoundTrip writes two granules of random scalefactors of every kind of block, at
// every scalefac_compress and with every scfsi, and reads them back.
//
// Each granule must read as many bits as the builder wrote. The second granule's long bands in
// groups scfsi marks must hold the first granule's scalefactors.
func TestReadMPEG1RoundTrip(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(7, 8))

	for _, first := range blocks {
		for _, second := range blocks {
			for compress := range 16 {
				for scfsi := range 16 {
					reuse := [4]bool{scfsi&8 != 0, scfsi&4 != 0, scfsi&2 != 0, scfsi&1 != 0}
					name := first.name + " then " + second.name

					if second.info.BlockType == layer3.ShortBlocks || first.info.BlockType == layer3.ShortBlocks {
						if scfsi != 0 {
							continue
						}
					}

					roundTrip(t, rng, [2]block{first, second}, compress, reuse, name)
				}
			}
		}
	}
}

// TestReadMPEG1ScfsiAfterShortBlocks checks a second granule of long blocks that reuses every group
// after a first granule of short blocks, which a stream should not send.
//
// Each long band then holds the scalefactor sent in its place in the first granule, as FFmpeg and
// mpg123 read it, and band 21 holds 0.
func TestReadMPEG1ScfsiAfterShortBlocks(t *testing.T) {
	t.Parallel()

	var (
		w       mp3test.BitWriter
		short   mp3test.Scalefactors
		side    layer3.SideInfo
		stream  bits.Reader
		factors layer3.Scalefactors
	)

	for band := range short.Short {
		for window := range short.Short[band] {
			short.Short[band][window] = uint32(band*3+window) % 8
		}
	}

	w.MPEG1Scalefactors(&mp3test.Granule{ScalefacCompress: 13, WindowSwitching: true, BlockType: 2}, [4]bool{}, &short)

	side.Granules[0][0] = layer3.GranuleInfo{ScalefacCompress: 13, BlockType: layer3.ShortBlocks}
	side.Granules[1][0] = layer3.GranuleInfo{ScalefacCompress: 13}
	side.SCFSI[0] = [4]bool{true, true, true, true}

	stream.Reset(w.Bytes(), 0)
	factors.ReadMPEG1(&stream, &side, 0, 0)
	factors.ReadMPEG1(&stream, &side, 1, 0)

	assert.Equal(t, w.Len(), stream.Pos())

	for band := range 21 {
		assert.Equal(t, int(short.Short[band/3][band%3]), factors.Long(band), "band %d", band)
	}

	assert.Zero(t, factors.Long(21))
}

// TestReadMPEG1EncodedFiles reads the scalefactors of every granule of MPEG-1 files LAME encoded,
// from each granule's start in its frame's main data.
//
// Each granule's scalefactors must take the published part2 length and fit in its part2_3_length.
func TestReadMPEG1EncodedFiles(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				side      layer3.SideInfo
				reservoir layer3.Reservoir
				stream    bits.Reader
				factors   [2]layer3.Scalefactors
			)

			for at, data := range encodedFrames(t, name) {
				head, _ := frame.Parse(data)
				require.Equal(t, frame.MPEG1, head.Version)
				require.NoError(t, side.Parse(head, data))

				main, missing := reservoir.MainData(side.MainDataBegin, data[head.DataAt():])
				require.Zero(t, missing)

				start := 0

				for gr := range 2 {
					for ch := range head.Channels() {
						info := side.Granules[gr][ch]

						stream.Reset(main, start)
						factors[ch].ReadMPEG1(&stream, &side, gr, ch)

						assert.Equal(t, part2Bits(info, side.SCFSI[ch], gr), stream.Pos()-start, "frame %d", at)
						assert.LessOrEqual(t, stream.Pos()-start, info.Part23Length, "frame %d", at)

						start += info.Part23Length
					}
				}
			}
		})
	}
}

// roundTrip writes two granules of random scalefactors and checks that they read back.
func roundTrip(t *testing.T, rng *rand.Rand, kinds [2]block, compress int, reuse [4]bool, name string) {
	t.Helper()

	var (
		w      mp3test.BitWriter
		side   layer3.SideInfo
		raw    [2]mp3test.Scalefactors
		marks  [2]int
		stream bits.Reader
	)

	side.SCFSI[0] = reuse

	for gr, kind := range kinds {
		raw[gr] = randomScalefactors(rng, slens[compress])
		kind.raw.ScalefacCompress = uint32(compress)
		kind.info.ScalefacCompress = compress
		side.Granules[gr][0] = kind.info

		groups := [4]bool{}
		if gr == 1 {
			groups = reuse
		}

		w.MPEG1Scalefactors(&kind.raw, groups, &raw[gr])
		marks[gr] = w.Len()
	}

	var factors layer3.Scalefactors

	stream.Reset(w.Bytes(), 0)

	for gr, kind := range kinds {
		factors.ReadMPEG1(&stream, &side, gr, 0)
		require.Equal(t, marks[gr], stream.Pos(), "%s granule %d bits", name, gr)

		want := raw[gr]

		if gr == 1 {
			for group, bands := range [4][2]int{{0, 6}, {6, 11}, {11, 16}, {16, 21}} {
				if reuse[group] {
					copy(want.Long[bands[0]:bands[1]], raw[0].Long[bands[0]:bands[1]])
				}
			}
		}

		assertScalefactors(t, kind.info, &want, &factors, name)
	}
}

// assertScalefactors checks that read scalefactors hold the raw ones of a kind of block, and 0 for
// the last band.
func assertScalefactors(t *testing.T, info layer3.GranuleInfo, want *mp3test.Scalefactors, got *layer3.Scalefactors, name string) {
	t.Helper()

	if info.BlockType != layer3.ShortBlocks {
		for band, factor := range want.Long {
			assert.Equal(t, int(factor), got.Long(band), "%s long band %d", name, band)
		}

		assert.Zero(t, got.Long(21), name)

		return
	}

	from := 0

	if info.MixedBlock {
		for band := range 8 {
			assert.Equal(t, int(want.Long[band]), got.Long(band), "%s long band %d", name, band)
		}

		from = 3
	}

	for band := from; band < 12; band++ {
		for window, factor := range want.Short[band] {
			assert.Equal(t, int(factor), got.Short(band, window), "%s short band %d window %d", name, band, window)
		}
	}

	for window := range 3 {
		assert.Zero(t, got.Short(12, window), name)
	}
}

// randomScalefactors returns random scalefactors within the widths of slen1 and slen2.
func randomScalefactors(rng *rand.Rand, slen [2]int) mp3test.Scalefactors {
	var factors mp3test.Scalefactors

	for band := range factors.Long {
		factors.Long[band] = rng.Uint32N(1 << slen[min(band/11, 1)])
	}

	for band := range factors.Short {
		for window := range factors.Short[band] {
			factors.Short[band][window] = rng.Uint32N(1 << slen[min(band/6, 1)])
		}
	}

	return factors
}

// part2Bits returns the published number of bits a granule's MPEG-1 scalefactors take.
func part2Bits(info layer3.GranuleInfo, scfsi [4]bool, gr int) int {
	slen := slens[info.ScalefacCompress]

	switch {
	case info.BlockType == layer3.ShortBlocks && info.MixedBlock:
		return 17*slen[0] + 18*slen[1]
	case info.BlockType == layer3.ShortBlocks:
		return 18 * (slen[0] + slen[1])
	default:
		bits := 0

		for group, bands := range [4]int{6, 5, 5, 5} {
			if gr == 0 || !scfsi[group] {
				bits += bands * slen[group/2]
			}
		}

		return bits
	}
}
