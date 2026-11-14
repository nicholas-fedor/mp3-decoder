// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readMask selects the header bits Parse reads.
//
// It leaves out the private bit, copyright, original and emphasis.
const readMask = 0xfffffef0

// FuzzParse checks that an accepted header re-encodes to the bits Parse read, and that its frame
// is large enough to hold its own header, CRC and side information.
//
// The seed corpus holds one header of each version and layer, free format and a header with no
// sync word.
func FuzzParse(f *testing.F) {
	for _, head := range [][]byte{
		{0xff, 0xfb, 0x90, 0x00},
		encode(fields{version: rawMPEG2, layer: rawLayer3, bitrate: 9, mode: 1, ext: 2, padding: true}),
		encode(fields{version: rawMPEG25, layer: rawLayer3, bitrate: 1, rate: 2, mode: 3}),
		encode(fields{version: rawMPEG1, layer: rawLayer1, bitrate: 14, rate: 1}),
		encode(fields{version: rawMPEG2, layer: rawLayer2, bitrate: 1, rate: 1}),
		encode(fields{version: rawMPEG1, layer: rawLayer3}),
		{0xff, 0xdb, 0x90, 0x00},
	} {
		f.Add(head)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		head, ok := Parse(b)
		if !ok {
			return
		}

		again, ok := Parse(reencode(head))
		require.True(t, ok)
		assert.Equal(t, head, again)
		assert.Equal(t, binary.BigEndian.Uint32(b)&readMask, binary.BigEndian.Uint32(reencode(head))&readMask)

		if size := head.Size(); size != 0 {
			assert.GreaterOrEqual(t, size, head.DataAt())
		}
	})
}

// reencode builds the header bytes for h.
func reencode(h Header) []byte {
	version := map[Version]uint32{MPEG1: rawMPEG1, MPEG2: rawMPEG2, MPEG25: rawMPEG25}[h.Version]
	layer := map[Layer]uint32{Layer1: rawLayer1, Layer2: rawLayer2, Layer3: rawLayer3}[h.Layer]

	table := lsfBitrates[h.Layer]
	if h.Version == MPEG1 {
		table = mpeg1Bitrates[h.Layer]
	}

	f := fields{
		version: version, layer: layer, ext: uint32(h.Extension), mode: uint32(h.Mode) - 1,
		padding: h.Padding, unprotected: !h.Protected,
	}

	for index, kbps := range table {
		if kbps*kilo == h.Bitrate {
			f.bitrate = uint32(index)
		}
	}

	for index, rate := range sampleRates[h.Version] {
		if rate == h.SampleRate {
			f.rate = uint32(index)
		}
	}

	return encode(f)
}
