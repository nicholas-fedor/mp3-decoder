// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDigest checks the hash against SHA-256 of the samples' bytes written out by hand, and that
// samples that compare equal but differ in their bits hash apart.
func TestDigest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		decoded [][2]float64
		bytes   string
	}{
		{"no samples", nil, ""},
		{"one pair", [][2]float64{{1, -0.5}}, "000000000000f03f000000000000e0bf"},
		{"two pairs", [][2]float64{{0, 0}, {-2, 0.25}}, "0000000000000000000000000000000000000000000000c0000000000000d03f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := hex.DecodeString(tt.bytes)
			require.NoError(t, err)
			assert.Equal(t, sha256.Sum256(data), Digest(tt.decoded))
		})
	}

	assert.NotEqual(t, Digest([][2]float64{{0, 0}}), Digest([][2]float64{{math.Copysign(0, -1), 0}}))
}
