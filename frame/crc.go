// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

// CRC-16 parameters.
const (
	// crcInit is the initial state of the CRC register.
	crcInit = 0xffff
	// crcPolynomial is the CRC polynomial x^16 + x^15 + x^2 + 1, without its top term.
	crcPolynomial = 0x8005
	// crcHeaderAt is the offset of the header bytes the CRC covers.
	crcHeaderAt = 2
	// crcTop is the top bit of the CRC register.
	crcTop = 15
)

// CheckCRC reports whether the CRC of the frame that h heads matches the bits it covers.
//
// The CRC covers the last two header bytes and the first bits after the CRC. In Layer I those bits
// are the bit allocation, and in Layer III the side information. A frame without a CRC passes.
//
// Parameters:
//   - frame: the frame's bytes from its header onward.
//   - bits: the number of bits after the CRC that it covers.
//
// Returns:
//   - bool: false when the CRC does not match or frame ends before the covered bits do.
func (h Header) CheckCRC(frame []byte, bits int) bool {
	if !h.Protected {
		return true
	}

	data := HeaderSize + crcSize
	if bits < 0 || len(frame)*bitsPerByte < data*bitsPerByte+bits {
		return false
	}

	crc := crcUpdate(crcInit, frame[crcHeaderAt:HeaderSize], (HeaderSize-crcHeaderAt)*bitsPerByte)

	crc = crcUpdate(crc, frame[data:], bits)

	return crc == uint16(frame[HeaderSize])<<bitsPerByte|uint16(frame[HeaderSize+1])
}

// crcUpdate feeds bits into a CRC register, most significant bit first.
//
// Parameters:
//   - crc: the register.
//   - data: the bytes that hold the bits.
//   - bits: the number of bits to feed.
//
// Returns:
//   - uint16: the register after the bits.
func crcUpdate(crc uint16, data []byte, bits int) uint16 {
	for at := range bits {
		bit := uint16(data[at/bitsPerByte] >> (bitsPerByte - 1 - at%bitsPerByte) & 1)
		top := crc >> crcTop

		crc <<= 1
		if top != bit {
			crc ^= crcPolynomial
		}
	}

	return crc
}
