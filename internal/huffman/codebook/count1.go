// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package codebook

// count1BLen is the length of every code word of count1 table B, which is its value's 4 bits
// inverted.
const count1BLen = 4

// count1A holds the code words of count1 table A, at v<<3 | w<<2 | x<<1 | y.
var count1A = [Count1Values]uint32{
	0b1_1, 0b1_0101, 0b1_0100, 0b1_00101,
	0b1_0110, 0b1_000101, 0b1_00100, 0b1_000100,
	0b1_0111, 0b1_00011, 0b1_00110, 0b1_000000,
	0b1_00111, 0b1_000010, 0b1_000011, 0b1_000001,
}

// count1B returns the code word of a value in count1 table B.
//
// Parameters:
//   - value: the value, v<<3 | w<<2 | x<<1 | y.
//
// Returns:
//   - Code: the value's 4 bits, inverted.
func count1B(value int) Code {
	return Code{Bits: uint32(^value & (Count1Values - 1)), Len: count1BLen}
}
