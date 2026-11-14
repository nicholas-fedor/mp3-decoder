// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBigValues checks the bits of pairs in tables with and without linbits against the published
// code words.
func TestBigValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		table  int
		values []int
		bits   string
	}{
		{"zeros take no sign", 1, []int{0, 0}, "1"},
		{"x of 1, y of 0", 1, []int{1, 0}, "01" + "0"},
		{"negative y", 1, []int{0, -1}, "001" + "1"},
		{"both signed", 1, []int{-1, 1}, "000" + "1" + "0"},
		{"table 0 writes nothing", 0, []int{0, 0}, ""},
		{"escape with linbits", 16, []int{16, 0}, "000001100" + "1" + "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var w BitWriter

			w.BigValues(tt.table, tt.values)

			assert.Equal(t, tt.bits, bitString(w))
		})
	}
}

// TestCount1 checks the bits of quadruples in tables A and B.
func TestCount1(t *testing.T) {
	t.Parallel()

	var a BitWriter

	a.Count1(0, []int{0, 0, 0, 0, 1, 0, 0, -1})
	assert.Equal(t, "1"+"00011"+"0"+"1", bitString(a))

	var b BitWriter

	b.Count1(1, []int{0, 0, 0, 0, -1, 1, 1, 1})
	assert.Equal(t, "1111"+"0000"+"1000", bitString(b))
}

// TestUnusedTable checks that an unused table writes nothing.
func TestUnusedTable(t *testing.T) {
	t.Parallel()

	var w BitWriter

	w.BigValues(4, []int{0, 0})
	w.Count1(2, []int{0, 0, 0, 0})

	assert.Zero(t, w.Len())
}

// bitString returns the bits a writer wrote as ones and zeros.
func bitString(w BitWriter) string {
	out := make([]byte, 0, w.Len())

	for i := range w.Len() {
		out = append(out, '0'+w.Bytes()[i/8]>>(7-i%8)&1)
	}

	return string(out)
}
