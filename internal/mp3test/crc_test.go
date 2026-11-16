// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtect checks the CRC against the published check value of this CRC-16, the CRC of the
// ASCII digits 1 to 9, and against the CRC of a frame LAME encoded.
func TestProtect(t *testing.T) {
	t.Parallel()

	digits := slices.Concat([]byte{0xff, 0xfa}, []byte("12"), []byte{0, 0}, []byte("3456789"))
	Protect(digits, 7*byteBits)
	assert.Equal(t, []byte{0xae, 0xe7}, digits[crcAt:crcAt+crcSize])

	data, err := os.ReadFile("../../testdata/protected.mp3")
	require.NoError(t, err)

	// The file opens with an MPEG-1 stereo frame, whose CRC covers 32 bytes of side information.
	encoded := data[:crcAt+crcSize+32]
	cleared := slices.Clone(encoded)
	cleared[crcAt], cleared[crcAt+1] = 0, 0

	assert.Equal(t, encoded, Protect(cleared, 32*byteBits))
}
