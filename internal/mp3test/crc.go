// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3test

// CRC layout.
const (
	// crcPolynomial is the CRC-16 polynomial x^16 + x^15 + x^2 + 1, its top term included.
	crcPolynomial = 0x18005
	// crcAt is the offset of the CRC in a protected frame.
	crcAt = headerSize
	// crcSize is the size of the CRC in bytes.
	crcSize = 2
	// crcCovered is the offset of the header bytes the CRC covers.
	crcCovered = 2
)

// Protect writes the CRC of a protected frame.
//
// The CRC covers the last two header bytes and the first bits after the CRC. It is the remainder
// of those bits, with the header bits complemented and 16 zeros after them, divided by the CRC
// polynomial.
//
// Parameters:
//   - frame: the frame from its header on, with room for the CRC.
//   - bits: the number of bits after the CRC that the CRC covers.
//
// Returns:
//   - []byte: frame.
func Protect(frame []byte, bits int) []byte {
	covered := make([]int, 0, 2*byteBits*crcSize+bits)

	for at := range byteBits * crcSize {
		covered = append(covered, 1-bitAt(frame[crcCovered:], at))
	}

	for at := range bits {
		covered = append(covered, bitAt(frame[crcAt+crcSize:], at))
	}

	covered = append(covered, make([]int, byteBits*crcSize)...)

	remainder := 0

	for _, bit := range covered {
		remainder = remainder<<1 | bit
		if remainder>>(byteBits*crcSize) == 1 {
			remainder ^= crcPolynomial
		}
	}

	frame[crcAt], frame[crcAt+1] = byte(remainder>>byteBits), byte(remainder)

	return frame
}

// bitAt returns a bit of some bytes, counting from the highest bit of the first.
//
// Parameters:
//   - data: the bytes.
//   - at: the bit's position.
//
// Returns:
//   - int: the bit.
func bitAt(data []byte, at int) int {
	return int(data[at/byteBits] >> (byteBits - 1 - at%byteBits) & 1)
}
