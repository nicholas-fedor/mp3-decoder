// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
)

// Stream is a conformance stream and the references its decoded output is measured against.
type Stream struct {
	// Name is the stream's file name without its extension.
	Name string
	// MD5 is the stream's MD5 checksum in hexadecimal, as ISO publishes it.
	MD5 string
	// References holds the references, which together cover the decoded output.
	References []Reference
	// MP4 reports whether ISO publishes the stream inside an MP4 file, as it does the Layer I and
	// Layer II streams, rather than as MPEG audio frames alone.
	MP4 bool
}

// Reference is a reference waveform and the part of a stream's decoded output it covers.
type Reference struct {
	// Name is the reference's file name without its .wav extension.
	Name string
	// MD5 is the reference's MD5 checksum in hexadecimal, as ISO publishes it.
	MD5 string
	// Channels holds the decoded channel that each channel of the reference is measured against,
	// 0 for left and 1 for right.
	Channels []int
	// Start is the decoded sample that the reference's first sample matches.
	Start int
	// Alternative reports whether the reference is an alternative to the reference before it, so
	// that matching either one conforms.
	Alternative bool
}

// file is a file in one of ISO's conformance archives.
type file struct {
	// archive is the archive's file name.
	archive string
	// member is the file's path in the archive, below the archive's top directory.
	member string
	// md5 is the file's MD5 checksum in hexadecimal.
	md5 string
}

// ArchiveURL is where ISO publishes the conformance archives of ISO/IEC 14496-26.
const ArchiveURL = "https://standards.iso.org/iso-iec/14496/-26/ed-2/en/"

// Where the archives keep the streams and the references.
const (
	// streamArchive is the archive of the MPEG-1 and MPEG-2 audio streams.
	streamArchive = "compressedMpeg12.zip"
	// streamDir is the streams' directory in their archive.
	streamDir = "compressedMpeg12/add-opt/"
	// mp4Archive is the archive of the streams in MP4 files.
	mp4Archive = "compressedMp4.zip"
	// mp4Dir is the MP4 files' directory in their archive.
	mp4Dir = "compressedMp4/"
	// referenceArchive is the archive of the reference waveforms.
	referenceArchive = "referencesWav.zip"
	// referenceDir is the references' directory in their archive.
	referenceDir = "referencesWav/"
)

// Streams returns the conformance streams.
//
// Returns:
//   - []Stream: the streams, by name.
func Streams() []Stream {
	listed := slices.Concat(layer12Streams, streams)

	for at := range listed {
		listed[at].References = slices.Clone(listed[at].References)

		for ref := range listed[at].References {
			listed[at].References[ref].Channels = slices.Clone(listed[at].References[ref].Channels)
		}
	}

	return listed
}

// Path returns where the stream lies in a cache.
//
// Parameters:
//   - dir: the cache.
//
// Returns:
//   - string: the stream's path.
func (s Stream) Path(dir string) string {
	return s.file().path(dir)
}

// Read returns the stream's MPEG audio frames from a cache.
//
// A stream that ISO publishes inside an MP4 file is taken out of it.
//
// Parameters:
//   - dir: the cache.
//
// Returns:
//   - []byte: the frames.
//   - error: the read failure, or [ErrMP4] for an MP4 file whose samples cannot be read.
func (s Stream) Read(dir string) ([]byte, error) {
	data, err := os.ReadFile(s.Path(dir))
	if err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	if !s.MP4 {
		return data, nil
	}

	return mp4Frames(data)
}

// file returns the stream's file in its archive.
//
// Returns:
//   - file: the file.
func (s Stream) file() file {
	if s.MP4 {
		return file{archive: mp4Archive, member: mp4Dir + s.Name + ".mp4", md5: s.MD5}
	}

	return file{archive: streamArchive, member: streamDir + s.Name + ".mpg", md5: s.MD5}
}

// Path returns where the reference lies in a cache.
//
// Parameters:
//   - dir: the cache.
//
// Returns:
//   - string: the reference's path.
func (r Reference) Path(dir string) string {
	return r.file().path(dir)
}

// file returns the reference's file in its archive.
//
// Returns:
//   - file: the file.
func (r Reference) file() file {
	return file{archive: referenceArchive, member: referenceDir + r.Name + ".wav", md5: r.MD5}
}

// path returns where the file lies in a cache, which holds every file under its own name.
//
// Parameters:
//   - dir: the cache.
//
// Returns:
//   - string: the file's path.
func (f file) path(dir string) string {
	return filepath.Join(dir, path.Base(f.member))
}
