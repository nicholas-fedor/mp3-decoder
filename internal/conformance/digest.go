// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// Digest returns the SHA-256 hash of decoded samples.
//
// It hashes the bits of each sample as a little-endian float64, left then right, pair by pair. Two
// decodes hash alike only when every bit of every sample agrees, so a hash recorded on one machine
// checks the output of every other.
//
// Parameters:
//   - decoded: the samples, as left and right pairs.
//
// Returns:
//   - [sha256.Size]byte: the hash.
func Digest(decoded [][2]float64) [sha256.Size]byte {
	hash := sha256.New()

	var pair []byte

	for _, samples := range decoded {
		pair = binary.LittleEndian.AppendUint64(pair[:0], math.Float64bits(samples[0]))
		pair = binary.LittleEndian.AppendUint64(pair, math.Float64bits(samples[1]))

		hash.Write(pair) //nolint:revive // hash.Hash documents that Write never returns an error.
	}

	return [sha256.Size]byte(hash.Sum(nil))
}
