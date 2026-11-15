// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamsCopies checks that changing the streams Streams returns leaves the next call's
// streams as they were, down to each reference's channels.
func TestStreamsCopies(t *testing.T) {
	t.Parallel()

	changed := Streams()
	changed[0].Name = "changed"
	changed[0].References[0].Name = "changed"
	changed[0].References[0].Channels[0] = 9

	fresh := Streams()
	assert.Equal(t, "l3_10101", fresh[0].Name)
	assert.Equal(t, "l3_10101", fresh[0].References[0].Name)
	assert.Equal(t, []int{0}, fresh[0].References[0].Channels)
}

// TestPaths checks that a cache holds each stream and reference under its own file name.
func TestPaths(t *testing.T) {
	t.Parallel()

	stream := Streams()[0]
	require.NotEmpty(t, stream.References)

	assert.Equal(t, filepath.Join("cache", "l3_10101.mpg"), stream.Path("cache"))
	assert.Equal(t, filepath.Join("cache", "l3_10101.wav"), stream.References[0].Path("cache"))
	assert.Equal(t, file{archive: streamArchive, member: streamDir + "l3_10101.mpg", md5: stream.MD5}, stream.file())
}
