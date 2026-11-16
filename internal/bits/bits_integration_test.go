// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package bits_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestRoundTrip reads back the fields of an MPEG-1 side information layout that a bit writer
// wrote.
func TestRoundTrip(t *testing.T) {
	t.Parallel()

	fields := []struct {
		value uint32
		width int
	}{
		{300, 9},
		{0b10101, 5},
		{0b1001, 4},
		{0b0110, 4},
		{2000, 12},
		{288, 9},
		{210, 8},
		{9, 4},
		{0, 1},
		{31, 5},
		{0, 5},
		{17, 5},
		{15, 4},
		{7, 3},
		{1, 1},
		{0, 1},
		{1, 1},
		{0xdeadbeef, 32},
	}

	var w mp3test.BitWriter

	for _, f := range fields {
		w.Write(f.value, f.width)
	}

	var r bits.Reader

	r.Reset(w.Bytes(), 0)

	for _, f := range fields {
		assert.Equal(t, f.value, r.Read(f.width), "%d bits", f.width)
	}

	assert.Equal(t, w.Len(), r.Pos())
}
