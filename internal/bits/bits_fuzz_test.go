// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package bits

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzReader reads fields whose widths come from the input and checks each against the model.
//
// The seed corpus holds empty input, one byte and a run of widths over random bytes.
func FuzzReader(f *testing.F) {
	f.Add([]byte{}, []byte{}, uint16(0))
	f.Add([]byte{0xa5}, []byte{3, 5, 8}, uint16(1))
	f.Add(noise(64), []byte{1, 9, 19, 32, 0, 13, 7}, uint16(37))

	f.Fuzz(func(t *testing.T, data, widths []byte, start uint16) {
		var r Reader

		pos := int(start)
		r.Reset(data, pos)

		for _, width := range widths {
			n := int(width) % (MaxRead + 1)

			require.Equal(t, model(data, pos, n), r.Read(n))

			pos += n
			require.Equal(t, pos, r.Pos())
		}
	})
}
