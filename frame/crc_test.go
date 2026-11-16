// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

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

// TestCheckCRC flips covered bits of a protected frame and cuts it short.
func TestCheckCRC(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/protected.mp3")
	require.NoError(t, err)

	head, ok := Parse(data)
	require.True(t, ok)

	frame := data[:head.Size()]
	bits := head.SideInfoSize() * bitsPerByte

	require.True(t, head.CheckCRC(frame, bits))

	for _, flip := range []int{crcHeaderAt, head.DataAt() - 1} {
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

	head, ok := Parse(mpeg1Head.Bytes())
	require.True(t, ok)

	assert.True(t, head.CheckCRC(nil, 0))
}
