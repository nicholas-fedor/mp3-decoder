// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamList checks the stream list: names that are unique, checksums of 16 bytes, MP4 files
// for the Layer I and Layer II streams alone, and references that take one or two distinct
// channels, start in order, and open with one that is not an alternative.
//
// A reference that two streams share must carry the same checksum in both.
func TestStreamList(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}
	checksums := map[string]string{}

	for _, stream := range Streams() {
		assert.False(t, names[stream.Name], "%s listed twice", stream.Name)
		assert.Equal(t, !strings.HasPrefix(stream.Name, "l3_"), stream.MP4, stream.Name)
		names[stream.Name] = true

		checkMD5(t, stream.Name, stream.MD5)
		require.NotEmpty(t, stream.References, stream.Name)
		assert.False(t, stream.References[0].Alternative, stream.Name)

		start := 0

		for _, ref := range stream.References {
			checkMD5(t, ref.Name, ref.MD5)

			if sum, ok := checksums[ref.Name]; ok {
				assert.Equal(t, sum, ref.MD5, ref.Name)
			}

			checksums[ref.Name] = ref.MD5

			assert.GreaterOrEqual(t, ref.Start, start, ref.Name)
			start = ref.Start

			require.NotEmpty(t, ref.Channels, ref.Name)
			assert.LessOrEqual(t, len(ref.Channels), 2, ref.Name)

			for at, channel := range ref.Channels {
				assert.Contains(t, []int{0, 1}, channel, ref.Name)
				assert.NotContains(t, ref.Channels[:at], channel, ref.Name)
			}
		}
	}

	assert.Len(t, names, 65)
}

// checkMD5 checks that a checksum is 16 bytes in lower-case hexadecimal.
func checkMD5(t *testing.T, name, sum string) {
	t.Helper()

	raw, err := hex.DecodeString(sum)
	require.NoError(t, err, name)
	assert.Len(t, raw, 16, name)
	assert.Equal(t, hex.EncodeToString(raw), sum, name)
}
