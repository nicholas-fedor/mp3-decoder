// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"encoding/binary"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// ignoredBits selects the header bits Parse ignores.
//
// They are the private bit, copyright, original and emphasis.
const ignoredBits = 0x0000010f

// FuzzParse checks that the bits Parse ignores do not change its result, and that an accepted
// header's sizes and counts lie in range.
//
// The seed corpus holds one header of each version and layer, free format and a header with no
// sync word.
func FuzzParse(f *testing.F) {
	for _, head := range []mp3test.Header{
		mpeg1Head,
		{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 9, Mode: mp3test.JointStereo, Extension: 2, Padding: true},
		{Version: mp3test.MPEG25, Layer: mp3test.Layer3, Bitrate: 1, Rate: 2, Mode: mp3test.Mono, Protected: true},
		{Version: mp3test.MPEG1, Layer: mp3test.Layer1, Bitrate: 14, Rate: 1},
		{Version: mp3test.MPEG2, Layer: mp3test.Layer2, Bitrate: 1, Rate: 1},
		freeHead,
	} {
		f.Add(head.Bytes())
	}

	f.Add([]byte{0xff, 0xdb, 0x90, 0x00})

	f.Fuzz(func(t *testing.T, b []byte) {
		head, ok := Parse(b)
		if !ok {
			return
		}

		flipped := binary.BigEndian.AppendUint32(nil, binary.BigEndian.Uint32(b)^ignoredBits)
		again, ok := Parse(flipped)
		require.True(t, ok)
		assert.Equal(t, head, again)

		assert.Contains(t, []int{384, 576, 1152}, head.Samples())
		assert.Contains(t, []int{1, 2}, head.Channels())
		assert.Equal(t, head.Free, head.Size() == 0)

		if !head.Free {
			assert.GreaterOrEqual(t, head.Size(), head.DataAt())
		}
	})
}

// FuzzFind checks that a confirmed frame parses to the returned header and that a header of its
// stream follows it, and that every offset Find returns lies within the window.
//
// The seed corpus holds two frames, junk before them, a free-format stream and a window that ends
// before the second header.
func FuzzFind(f *testing.F) {
	two := slices.Concat(mpeg1Head.Frame(417), mpeg1Head.Frame(417))
	free := slices.Concat(freeHead.Frame(300), freeHead.Frame(300), freeHead.Frame(300))

	for _, window := range [][]byte{two, slices.Concat([]byte{0xff, 0xfe, 0x00}, two), free, two[:419]} {
		f.Add(window)
	}

	f.Fuzz(func(t *testing.T, window []byte) {
		at, head, found := Find(window, nil)

		require.GreaterOrEqual(t, at, 0)
		require.LessOrEqual(t, at, len(window))

		if !found {
			return
		}

		parsed, ok := Parse(window[at:])
		require.True(t, ok)
		assert.True(t, head.SameStream(parsed))
		require.Positive(t, head.Size())
		assert.Equal(t, confirmed, follows(window[at:], head, head.Size()))
	})
}
