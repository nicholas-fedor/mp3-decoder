// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"path/filepath"
	"slices"
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
	assert.Equal(t, "l1_fl1", fresh[0].Name)
	assert.Equal(t, "l1_fl1", fresh[0].References[0].Name)
	assert.Equal(t, []int{0, 1}, fresh[0].References[0].Channels)
}

// TestPaths checks that a cache holds each stream and reference under its own file name, and that
// the archive of a stream in an MP4 file is the MP4 archive.
func TestPaths(t *testing.T) {
	t.Parallel()

	streams := Streams()
	mp4 := streams[0]
	stream := streams[slices.IndexFunc(streams, func(s Stream) bool { return s.Name == "l3_10101" })]

	assert.Equal(t, filepath.Join("cache", "l3_10101.mpg"), stream.Path("cache"))
	assert.Equal(t, filepath.Join("cache", "l3_10101.wav"), stream.References[0].Path("cache"))
	assert.Equal(t, file{archive: streamArchive, member: streamDir + "l3_10101.mpg", md5: stream.MD5}, stream.file())

	assert.Equal(t, filepath.Join("cache", "l1_fl1.mp4"), mp4.Path("cache"))
	assert.Equal(t, file{archive: mp4Archive, member: mp4Dir + "l1_fl1.mp4", md5: mp4.MD5}, mp4.file())
}

// TestRead checks that a stream of frames reads as it is, and one in an MP4 file as its samples,
// and that a missing or malformed file fails.
func TestRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	frames := []byte("frames of MPEG audio")
	samples := [][]byte{[]byte("frames of "), []byte("MPEG audio")}

	require.NoError(t, os.WriteFile(filepath.Join(dir, "raw.mpg"), frames, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "packed.mp4"),
		mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 2}}, chunks: 1}), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.mp4"), []byte("not an MP4 file"), 0o600))

	got, err := Stream{Name: "raw"}.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, frames, got)

	got, err = Stream{Name: "packed", MP4: true}.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, frames, got)

	_, err = Stream{Name: "broken", MP4: true}.Read(dir)
	require.ErrorIs(t, err, ErrMP4)

	_, err = Stream{Name: "missing"}.Read(dir)
	require.ErrorIs(t, err, os.ErrNotExist)
}
