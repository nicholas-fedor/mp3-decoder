// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package bits

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// model reads n bits from data at pos one at a time, with zeros outside data.
func model(data []byte, pos, n int) uint32 {
	var value uint32

	for at := pos; at < pos+n; at++ {
		value <<= 1

		if at >= 0 && at/8 < len(data) {
			value |= uint32(data[at/8] >> (7 - at%8) & 1)
		}
	}

	return value
}

// noise returns n deterministic random bytes.
func noise(n int) []byte {
	rng := rand.New(rand.NewPCG(3, 4))
	data := make([]byte, n)

	for i := range data {
		data[i] = byte(rng.Uint32())
	}

	return data
}

// TestReaderMatchesModel checks every position and width against the model, including reads that
// run past the end of the bytes.
func TestReaderMatchesModel(t *testing.T) {
	t.Parallel()

	data := noise(24)

	var r Reader

	for pos := range len(data)*8 + 8 {
		for n := 0; n <= MaxRead; n++ {
			r.Reset(data, pos)

			require.Equal(t, model(data, pos, n), r.Peek(n), "peek %d at %d", n, pos)
			require.Equal(t, model(data, pos, n), r.Read(n), "read %d at %d", n, pos)
			require.Equal(t, pos+n, r.Pos())
		}
	}
}

// TestReaderSequence reads a run of fields of mixed widths, then skips.
func TestReaderSequence(t *testing.T) {
	t.Parallel()

	var r Reader

	r.Reset([]byte{0b1011_0011, 0b1100_0000, 0xff}, 0)

	assert.Equal(t, uint32(0b101), r.Read(3))
	assert.Equal(t, uint32(0b1_0011_11), r.Read(7))

	r.Skip(6)
	assert.Equal(t, 16, r.Pos())
	assert.Equal(t, uint32(0xff), r.Read(8))
	assert.Zero(t, r.Read(MaxRead))
}

// TestReaderOutside checks that positions before or past the bytes read as zeros.
func TestReaderOutside(t *testing.T) {
	t.Parallel()

	var r Reader

	r.Reset([]byte{0xff}, -16)
	assert.Zero(t, r.Peek(MaxRead))

	r.Reset(nil, 0)
	assert.Zero(t, r.Read(MaxRead))
}
