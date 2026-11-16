// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBitWriter checks the bytes and length of fields that cross byte boundaries.
func TestBitWriter(t *testing.T) {
	t.Parallel()

	var w BitWriter

	w.Write(0b101, 3)
	w.Write(0xff, 8)
	w.Write(0, 0)
	w.Write(0b1, 1)

	assert.Equal(t, 12, w.Len())
	assert.Equal(t, []byte{0b1011_1111, 0b1111_0000}, w.Bytes())
}
