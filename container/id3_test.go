// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestID3v2Size covers tags with and without a footer and the headers that are not tags.
func TestID3v2Size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head []byte
		want int
	}{
		{"tag", []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0x02, 0x01}, 10 + 257},
		{"tag with a footer", []byte{'I', 'D', '3', 4, 0, 0x10, 0, 0, 0x02, 0x01}, 20 + 257},
		{"largest size", []byte{'I', 'D', '3', 3, 0, 0, 0x7f, 0x7f, 0x7f, 0x7f}, 10 + 1<<28 - 1},
		{"size byte with its high bit set", []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0x80, 0}, 0},
		{"invalid version", []byte{'I', 'D', '3', 0xff, 0, 0, 0, 0, 0, 1}, 0},
		{"invalid revision", []byte{'I', 'D', '3', 4, 0xff, 0, 0, 0, 0, 1}, 0},
		{"no magic", []byte{'I', 'D', '4', 4, 0, 0, 0, 0, 0, 1}, 0},
		{"short", []byte{'I', 'D', '3', 4, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ID3v2Size(tt.head))
		})
	}
}
