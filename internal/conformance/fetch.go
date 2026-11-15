// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"archive/zip"
	"context"
	"crypto/md5" //nolint:gosec // ISO publishes MD5 checksums, which check the files for damage.
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ChecksumError is returned for a file whose MD5 checksum is not the one ISO publishes.
type ChecksumError struct {
	// Name is the file's name.
	Name string
	// Want is the checksum ISO publishes.
	Want string
	// Got is the file's checksum.
	Got string
}

// Limits of the cache.
const (
	// cacheMode is the permission of a cache directory that Fetch creates.
	cacheMode = 0o750
	// maxFileSize bounds the bytes taken from one member, far above the largest file's 3 MB.
	maxFileSize = 64 << 20
)

// ErrNoMember is returned for an archive that lacks a file it should hold.
var ErrNoMember = errors.New("conformance: archive lacks a file")

// Fetch makes sure that a cache holds every file of some streams, fetching what it lacks.
//
// A file already in the cache with the right checksum is kept, so Fetch sends no request when
// the cache is complete. Each file is written under a temporary name and renamed once its checksum
// is right, so the cache never holds a partial or damaged file under a file's own name.
//
// Parameters:
//   - ctx: cancels the requests.
//   - client: sends the requests.
//   - base: where the archives are published, such as [ArchiveURL].
//   - dir: the cache, created if it does not exist.
//   - streams: the streams whose files the cache must hold, with their references.
//
// Returns:
//   - int: the number of files fetched.
//   - error: a *[ChecksumError], [ErrNoMember], [ErrNoRange], or a request or file error.
func Fetch(ctx context.Context, client *http.Client, base, dir string, streams []Stream) (int, error) {
	err := os.MkdirAll(dir, cacheMode)
	if err != nil {
		return 0, fmt.Errorf("create cache: %w", err)
	}

	missing, archives, err := missingFiles(dir, streams)
	if err != nil {
		return 0, err
	}

	fetched := 0

	for _, archive := range archives {
		address, err := url.JoinPath(base, archive)
		if err != nil {
			return fetched, fmt.Errorf("archive URL: %w", err)
		}

		count, err := fetchArchive(ctx, client, address, dir, missing[archive])

		fetched += count

		if err != nil {
			return fetched, err
		}
	}

	return fetched, nil
}

// Error describes the mismatch.
//
// Returns:
//   - string: the description.
func (e *ChecksumError) Error() string {
	return fmt.Sprintf("conformance: %s has MD5 %s, not %s", e.Name, e.Got, e.Want)
}

// missingFiles lists the files of some streams that a cache lacks, or holds with another checksum.
//
// Parameters:
//   - dir: the cache.
//   - streams: the streams.
//
// Returns:
//   - map[string][]file: the missing files of each archive.
//   - []string: the archives with missing files, in the order the streams name them.
//   - error: a file error other than a missing file.
func missingFiles(dir string, streams []Stream) (map[string][]file, []string, error) {
	missing := map[string][]file{}
	seen := map[string]bool{}

	var archives []string

	for _, stream := range streams {
		files := []file{stream.file()}
		for _, ref := range stream.References {
			files = append(files, ref.file())
		}

		for _, want := range files {
			if seen[want.member] {
				continue
			}

			seen[want.member] = true

			sum, err := checksum(want.path(dir))

			switch {
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				return nil, nil, err
			case sum == want.md5:
				continue
			default:
			}

			if missing[want.archive] == nil {
				archives = append(archives, want.archive)
			}

			missing[want.archive] = append(missing[want.archive], want)
		}
	}

	return missing, archives, nil
}

// fetchArchive fetches files from a remote archive into a cache.
//
// Parameters:
//   - ctx: cancels the requests.
//   - client: sends the requests.
//   - address: the archive's URL.
//   - dir: the cache.
//   - files: the files to fetch.
//
// Returns:
//   - int: the number of files fetched.
//   - error: a *[ChecksumError], [ErrNoMember], [ErrNoRange], or a request or file error.
func fetchArchive(ctx context.Context, client *http.Client, address, dir string, files []file) (int, error) {
	archive, err := openRemote(ctx, client, address)
	if err != nil {
		return 0, err
	}

	// ISO's member names start with ../, which only matters to a reader that writes files under
	// their member names. Each file here is written under a name of its own.
	reader, err := zip.NewReader(archive, archive.size)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return 0, fmt.Errorf("read %s: %w", address, err)
	}

	for fetched, want := range files {
		index := -1

		for at, member := range reader.File {
			if member.Name == want.member || strings.HasSuffix(member.Name, "/"+want.member) {
				index = at
			}
		}

		if index < 0 {
			return fetched, fmt.Errorf("%w: %s in %s", ErrNoMember, want.member, address)
		}

		err = extract(reader.File[index], want, dir)
		if err != nil {
			return fetched, err
		}
	}

	return len(files), nil
}

// extract copies a member of an archive into a cache and checks its checksum.
//
// Parameters:
//   - member: the archive's member.
//   - want: the file the member holds.
//   - dir: the cache.
//
// Returns:
//   - error: a *[ChecksumError], or a read or file error.
func extract(member *zip.File, want file, dir string) error {
	source, err := member.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", want.member, err)
	}

	defer source.Close()

	temporary, err := os.CreateTemp(dir, ".fetch-*")
	if err != nil {
		return fmt.Errorf("create %s: %w", want.member, err)
	}

	defer os.Remove(temporary.Name())

	hash := md5.New() //nolint:gosec // ISO publishes MD5 checksums, which check the files for damage.

	// A member past the bound is cut short, so its checksum fails.
	_, err = io.CopyN(io.MultiWriter(temporary, hash), source, maxFileSize)
	if errors.Is(err, io.EOF) {
		err = nil
	}

	err = errors.Join(err, temporary.Close())
	if err != nil {
		return fmt.Errorf("fetch %s: %w", want.member, err)
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	if sum != want.md5 {
		return &ChecksumError{Name: want.member, Want: want.md5, Got: sum}
	}

	err = os.Rename(temporary.Name(), want.path(dir))
	if err != nil {
		return fmt.Errorf("store %s: %w", want.member, err)
	}

	return nil
}

// checksum returns the MD5 checksum of a file.
//
// Parameters:
//   - name: the file's path.
//
// Returns:
//   - string: the checksum in hexadecimal.
//   - error: a file error, matching [fs.ErrNotExist] for a missing file.
func checksum(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("check %s: %w", name, err)
	}

	sum := md5.Sum(data) //nolint:gosec // ISO publishes MD5 checksums, which check the files for damage.

	return hex.EncodeToString(sum[:]), nil
}
