// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// member is a file of a test archive.
type member struct {
	// name is the file's name in the archive.
	name string
	// data is the file's contents.
	data []byte
	// method is the compression method.
	method uint16
}

// archiveServer serves test archives under their names, running hook before each answer, and
// counts the requests.
type archiveServer struct {
	// server serves the archives.
	server *httptest.Server
	// hook runs before each answer, or is nil.
	hook func()
	// archives holds each archive's bytes by name.
	archives map[string][]byte
	// requests counts the requests.
	requests atomic.Int32
}

// Test files, named as the cache holds them.
var (
	// streamData is the contents of the test streams.
	streamData = map[string][]byte{"s1": []byte("first stream"), "s2": []byte("second stream")}
	// referenceData is the contents of the test references.
	referenceData = map[string][]byte{"r1": []byte("shared reference"), "r2": []byte("own reference")}
)

// sum returns the MD5 checksum of data in hexadecimal.
func sum(data []byte) string {
	digest := md5.Sum(data)

	return hex.EncodeToString(digest[:])
}

// testStreams returns two streams that share a reference.
func testStreams() []Stream {
	reference := func(name string) Reference {
		return Reference{Name: name, MD5: sum(referenceData[name]), Channels: []int{0}, Start: 0, Alternative: false}
	}

	return []Stream{
		{Name: "s1", MD5: sum(streamData["s1"]), References: []Reference{reference("r1"), reference("r2")}},
		{Name: "s2", MD5: sum(streamData["s2"]), References: []Reference{reference("r1")}},
	}
}

// zipOf returns a zip archive of the members.
func zipOf(t *testing.T, members ...member) []byte {
	t.Helper()

	var buffer bytes.Buffer

	writer := zip.NewWriter(&buffer)

	for _, file := range members {
		header := &zip.FileHeader{Name: file.name, Method: file.method}

		out, err := writer.CreateHeader(header)
		if file.method != zip.Store && file.method != zip.Deflate {
			out, err = writer.CreateRaw(header)
		}

		require.NoError(t, err)

		_, err = out.Write(file.data)
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

// isoArchives returns archives that hold the test files under ISO's layout, with other files
// beside them.
func isoArchives(t *testing.T) map[string][]byte {
	t.Helper()

	return map[string][]byte{
		streamArchive: zipOf(t,
			member{name: "../top/compressedMpeg12/", data: nil, method: zip.Store},
			member{name: "../top/compressedMpeg12/add-opt/s1.mpg", data: streamData["s1"], method: zip.Deflate},
			member{name: "../top/compressedMpeg12/add-opt/other.mpg", data: []byte("other"), method: zip.Deflate},
			member{name: "../top/compressedMpeg12/add-opt/s2.mpg", data: streamData["s2"], method: zip.Store},
		),
		referenceArchive: zipOf(t,
			member{name: "../top/referencesWav/r1.wav", data: referenceData["r1"], method: zip.Deflate},
			member{name: "../top/referencesWav/_checksum/r1.wav.md5sum", data: []byte("x"), method: zip.Deflate},
			member{name: "referencesWav/r2.wav", data: referenceData["r2"], method: zip.Deflate},
		),
	}
}

// serveArchives starts a server of archives.
func serveArchives(t *testing.T, archives map[string][]byte) *archiveServer {
	t.Helper()

	served := &archiveServer{server: nil, hook: nil, archives: archives, requests: atomic.Int32{}}
	served.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.requests.Add(1)

		if served.hook != nil {
			served.hook()
		}

		data, ok := served.archives[filepath.Base(r.URL.Path)]
		if !ok {
			http.NotFound(w, r)

			return
		}

		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
	}))
	t.Cleanup(served.server.Close)

	return served
}

// fetch fetches the test streams from a server into dir.
func (s *archiveServer) fetch(t *testing.T, dir string, streams []Stream) (int, error) {
	t.Helper()

	return Fetch(t.Context(), s.server.Client(), s.server.URL+"/iso/", dir, streams)
}

// assertCache checks that dir holds exactly the named files with their test contents.
func assertCache(t *testing.T, dir string, names ...string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	listed := make([]string, 0, len(entries))
	for _, entry := range entries {
		listed = append(listed, entry.Name())
	}

	assert.ElementsMatch(t, names, listed)

	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)

		ext := filepath.Ext(name)

		want := streamData[name[:len(name)-len(ext)]]
		if ext == ".wav" {
			want = referenceData[name[:len(name)-len(ext)]]
		}

		assert.Equal(t, want, data, name)
	}
}

// TestFetch fills an empty cache, checks that a complete cache sends no request, and checks that
// a damaged file is fetched again alone.
func TestFetch(t *testing.T) {
	t.Parallel()

	served := serveArchives(t, isoArchives(t))
	dir := filepath.Join(t.TempDir(), "cache", "mp3")

	fetched, err := served.fetch(t, dir, testStreams())
	require.NoError(t, err)
	assert.Equal(t, 4, fetched)
	assertCache(t, dir, "s1.mpg", "s2.mpg", "r1.wav", "r2.wav")

	requests := served.requests.Load()

	fetched, err = served.fetch(t, dir, testStreams())
	require.NoError(t, err)
	assert.Zero(t, fetched)
	assert.Equal(t, requests, served.requests.Load())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "r1.wav"), []byte("damaged"), 0o600))

	fetched, err = served.fetch(t, dir, testStreams())
	require.NoError(t, err)
	assert.Equal(t, 1, fetched)
	assertCache(t, dir, "s1.mpg", "s2.mpg", "r1.wav", "r2.wav")
}

// TestFetchInsecurePaths checks that ISO's member names, which start with ../, are read even when
// the archive reader refuses them by default.
func TestFetchInsecurePaths(t *testing.T) {
	t.Setenv("GODEBUG", "zipinsecurepath=0")

	served := serveArchives(t, isoArchives(t))
	dir := t.TempDir()

	fetched, err := served.fetch(t, dir, testStreams())
	require.NoError(t, err)
	assert.Equal(t, 4, fetched)
}

// TestFetchReadOnlyCache checks that a fetch into a cache it cannot write to fails and fetches
// nothing.
//
// Windows ignores the read-only attribute of a directory, so the test runs only where the cache
// can be made read-only.
func TestFetchReadOnlyCache(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Windows ignores the read-only attribute of a directory")
	}

	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	served := serveArchives(t, isoArchives(t))
	served.hook = func() { _ = os.Chmod(dir, 0o500) }

	fetched, err := served.fetch(t, dir, testStreams())
	require.Error(t, err)
	assert.Zero(t, fetched)

	require.NoError(t, os.Chmod(dir, 0o700))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// TestFetchErrors checks every way a fetch fails, the files it fetched before failing, and that no
// partial file stays in the cache.
//
// The hooks run while the archive is served, after the cache has been checked.
func TestFetchErrors(t *testing.T) {
	t.Parallel()

	stored := zipOf(t, member{name: "compressedMpeg12/add-opt/s1.mpg", data: streamData["s1"], method: zip.Store})
	damaged := bytes.Replace(stored, streamData["s1"], []byte("first strEam"), 1)

	tests := []struct {
		archives map[string][]byte
		prepare  func(t *testing.T, dir string)
		hook     func(dir string)
		check    func(t *testing.T, err error)
		name     string
		base     string
		streams  []Stream
		fetched  int
		cache    []string
	}{
		{
			name: "checksum", archives: isoArchives(t), base: "",
			streams: func() []Stream {
				changed := testStreams()
				changed[0].References[1].MD5 = sum([]byte("other"))

				return changed
			}(),
			prepare: nil, hook: nil, fetched: 3, cache: []string{"s1.mpg", "s2.mpg", "r1.wav"},
			check: func(t *testing.T, err error) {
				t.Helper()

				var checksumErr *ChecksumError

				require.ErrorAs(t, err, &checksumErr)
				assert.Equal(t, "referencesWav/r2.wav", checksumErr.Name)
				assert.Equal(t, sum(referenceData["r2"]), checksumErr.Got)
				assert.Contains(t, err.Error(), "referencesWav/r2.wav has MD5")
			},
		},
		{
			name: "missing member", base: "", streams: testStreams(), prepare: nil, hook: nil, fetched: 0, cache: nil,
			archives: map[string][]byte{streamArchive: zipOf(t, member{name: "s1.mpg", data: nil, method: zip.Store})},
			check:    errorIs(ErrNoMember),
		},
		{
			name: "missing archive", base: "", streams: testStreams(), prepare: nil, hook: nil, fetched: 2,
			archives: map[string][]byte{streamArchive: isoArchives(t)[streamArchive]},
			cache:    []string{"s1.mpg", "s2.mpg"}, check: errorIs(ErrNoRange),
		},
		{
			name: "not a zip", base: "", streams: testStreams(), prepare: nil, hook: nil, fetched: 0, cache: nil,
			archives: map[string][]byte{streamArchive: []byte("not a zip archive")}, check: errorIs(zip.ErrFormat),
		},
		{
			name: "unknown method", base: "", streams: testStreams()[:1], prepare: nil, hook: nil, fetched: 0, cache: nil,
			archives: map[string][]byte{streamArchive: zipOf(t, member{name: "compressedMpeg12/add-opt/s1.mpg", data: nil, method: 99})},
			check:    errorIs(zip.ErrAlgorithm),
		},
		{
			name: "damaged member", base: "", streams: testStreams()[:1], prepare: nil, hook: nil, fetched: 0, cache: nil,
			archives: map[string][]byte{streamArchive: damaged}, check: errorIs(zip.ErrChecksum),
		},
		{
			name: "bad URL", archives: isoArchives(t), base: "http://%zz/", streams: testStreams(),
			prepare: nil, hook: nil, fetched: 0, cache: nil, check: anyError,
		},
		{
			name: "cache entry is a directory", archives: isoArchives(t), base: "", streams: testStreams(), hook: nil,
			prepare: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.Mkdir(filepath.Join(dir, "r1.wav"), 0o700))
			},
			fetched: 0, cache: []string{"r1.wav"}, check: anyError,
		},
		{
			name: "file name taken", archives: isoArchives(t), base: "", streams: testStreams(), prepare: nil,
			hook:    func(dir string) { _ = os.MkdirAll(filepath.Join(dir, "s1.mpg", "taken"), 0o700) },
			fetched: 0, cache: []string{"s1.mpg"}, check: anyError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

			if test.prepare != nil {
				test.prepare(t, dir)
			}

			served := serveArchives(t, test.archives)
			if test.hook != nil {
				served.hook = func() { test.hook(dir) }
			}

			base := served.server.URL + "/iso/"
			if test.base != "" {
				base = test.base
			}

			fetched, err := Fetch(t.Context(), served.server.Client(), base, dir, test.streams)
			test.check(t, err)
			assert.Equal(t, test.fetched, fetched)

			require.NoError(t, os.Chmod(dir, 0o700))

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)

			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				names = append(names, entry.Name())
			}

			assert.ElementsMatch(t, test.cache, names)
		})
	}
}

// TestFetchCacheIsAFile checks that a cache path naming a file is refused before any request.
func TestFetchCacheIsAFile(t *testing.T) {
	t.Parallel()

	served := serveArchives(t, isoArchives(t))
	dir := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(dir, nil, 0o600))

	_, err := served.fetch(t, dir, testStreams())
	require.Error(t, err)
	assert.Zero(t, served.requests.Load())
}

// errorIs returns a check that an error matches target.
func errorIs(target error) func(t *testing.T, err error) {
	return func(t *testing.T, err error) {
		t.Helper()
		require.ErrorIs(t, err, target)
	}
}

// anyError checks that there is an error.
func anyError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
}
