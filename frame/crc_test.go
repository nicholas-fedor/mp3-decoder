// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCRCUpdate checks the register against the published check value of this CRC-16, the CRC of
// the ASCII digits 1 to 9.
func TestCRCUpdate(t *testing.T) {
	t.Parallel()

	data := []byte("123456789")

	assert.Equal(t, uint16(0xaee7), crcUpdate(crcInit, data, len(data)*bitsPerByte))
}

// TestCheckCRC checks every frame of a CRC-protected file, then flips covered bits and cuts frames
// short.
func TestCheckCRC(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/protected.mp3")
	require.NoError(t, err)

	for at := 0; at < len(data); {
		head, ok := Parse(data[at:])
		require.True(t, ok)
		require.True(t, head.Protected)

		frame := data[at : at+head.Size()]
		bits := head.SideInfoSize() * bitsPerByte

		assert.True(t, head.CheckCRC(frame, bits), "frame at %d", at)

		at += head.Size()
	}

	head, _ := Parse(data)
	frame := data[:head.Size()]
	bits := head.SideInfoSize() * bitsPerByte

	for _, flip := range []int{2, head.DataAt() - 1} {
		broken := slices.Clone(frame)
		broken[flip] ^= 1

		assert.False(t, head.CheckCRC(broken, bits), "byte %d flipped", flip)
	}

	assert.False(t, head.CheckCRC(frame[:head.DataAt()-1], bits))
	assert.False(t, head.CheckCRC(frame, -1))
}

// TestCheckCRCUnprotected checks that a frame without a CRC passes.
func TestCheckCRCUnprotected(t *testing.T) {
	t.Parallel()

	head, ok := Parse(encode(mpeg1Frame))
	require.True(t, ok)

	assert.True(t, head.CheckCRC(nil, 0))
}
