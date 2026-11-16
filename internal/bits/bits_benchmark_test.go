// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package bits

import "testing"

// BenchmarkRead measures reads of the widths Layer III fields and Huffman codes take, through
// 4 KiB of bits.
func BenchmarkRead(b *testing.B) {
	data := noise(4 << 10)
	widths := []int{1, 4, 9, 12, 19, 3, 13, 8, 2, 5}

	b.SetBytes(int64(len(data)))

	var r Reader

	for b.Loop() {
		r.Reset(data, 0)

		for r.Pos() < len(data)*8 {
			for _, n := range widths {
				r.Read(n)
			}
		}
	}
}
