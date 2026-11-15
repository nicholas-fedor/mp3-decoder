// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"encoding/binary"
	"errors"
)

// MP4 layout.
const (
	// boxHeaderSize is the size of a box's 32-bit size and its type.
	boxHeaderSize = 8
	// largeHeaderSize is the size of a box header that carries a 64-bit size.
	largeHeaderSize = 16
	// fullBoxSize is the size of a full box's version and flags.
	fullBoxSize = 4
	// stszHeaderSize is the size of a stsz box's fields before its sizes: the version and flags,
	// the size every sample takes, and the number of samples.
	stszHeaderSize = 12
	// stscEntrySize is the size of a stsc entry: its first chunk, its samples per chunk and its
	// sample description.
	stscEntrySize = 12
	// countSize is the size of a table's entry count.
	countSize = 4
	// largeSize is the 32-bit size of a box whose 64-bit size follows its type.
	largeSize = 1
	// offsetBytes is the size of a chunk offset in stco.
	offsetBytes = 4
	// largeOffsetBytes is the size of a chunk offset in co64.
	largeOffsetBytes = 8
)

// ErrMP4 is returned for an MP4 file whose first track's samples cannot be found.
var ErrMP4 = errors.New("conformance: not an MP4 file whose samples can be read")

// mp4Frames returns the samples of an MP4 file's first track, end to end.
//
// ISO publishes its Layer I and Layer II streams in MP4 files that hold each MPEG audio frame as
// one sample, so their samples end to end are the stream. The sample table gives each sample's
// size, the chunks that hold the samples, and where each chunk starts.
//
// Parameters:
//   - data: the MP4 file.
//
// Returns:
//   - []byte: the samples.
//   - error: [ErrMP4] when the sample table is missing, malformed, or names bytes past the file.
func mp4Frames(data []byte) ([]byte, error) {
	table, ok := box(data, "moov", "trak", "mdia", "minf", "stbl")
	if !ok {
		return nil, ErrMP4
	}

	sizes, ok := sampleSizes(table, uint64(len(data)))
	if !ok {
		return nil, ErrMP4
	}

	offsets, ok := chunkOffsets(table)
	if !ok {
		return nil, ErrMP4
	}

	runs, ok := sampleRuns(table)
	if !ok {
		return nil, ErrMP4
	}

	return joinSamples(data, sizes, offsets, runs)
}

// joinSamples copies the samples out of their chunks, end to end.
//
// Parameters:
//   - data: the MP4 file.
//   - sizes: each sample's size.
//   - offsets: where each chunk starts.
//   - runs: the stsc entries.
//
// Returns:
//   - []byte: the samples.
//   - error: [ErrMP4] when the chunks hold other than the samples, or name bytes past the file.
func joinSamples(data []byte, sizes []uint32, offsets []uint64, runs []byte) ([]byte, error) {
	var out []byte

	sample := 0

	for chunk, offset := range offsets {
		for range samplesInChunk(runs, chunk+1) {
			if sample == len(sizes) || offset > uint64(len(data)) || uint64(sizes[sample]) > uint64(len(data))-offset {
				return nil, ErrMP4
			}

			out = append(out, data[offset:offset+uint64(sizes[sample])]...)

			offset += uint64(sizes[sample])
			sample++
		}
	}

	if sample != len(sizes) {
		return nil, ErrMP4
	}

	return out, nil
}

// box returns the body of the box that a path of box types names, each inside the one before.
//
// Parameters:
//   - data: the boxes to search.
//   - path: the box types, outermost first.
//
// Returns:
//   - []byte: the body of the last box of the path.
//   - bool: whether the path was found whole.
func box(data []byte, path ...string) ([]byte, bool) {
	for _, kind := range path {
		var found bool

		data, found = child(data, kind)
		if !found {
			return nil, false
		}
	}

	return data, true
}

// child returns the body of the first box of a type among a run of boxes.
//
// Parameters:
//   - data: the run of boxes.
//   - kind: the box type.
//
// Returns:
//   - []byte: the box's body.
//   - bool: whether a whole box of the type was found.
func child(data []byte, kind string) ([]byte, bool) {
	for len(data) >= boxHeaderSize {
		size, header := uint64(binary.BigEndian.Uint32(data)), uint64(boxHeaderSize)

		switch size {
		case 0:
			size = uint64(len(data))
		case largeSize:
			if len(data) < largeHeaderSize {
				return nil, false
			}

			size, header = binary.BigEndian.Uint64(data[boxHeaderSize:]), largeHeaderSize
		default:
		}

		if size < header || size > uint64(len(data)) {
			return nil, false
		}

		if string(data[4:boxHeaderSize]) == kind {
			return data[header:size], true
		}

		data = data[size:]
	}

	return nil, false
}

// sampleSizes reads the size of every sample from a sample table's stsz box.
//
// Parameters:
//   - table: the sample table's body.
//   - limit: the size of the file, which the samples must fit.
//
// Returns:
//   - []uint32: each sample's size.
//   - bool: whether the box was found whole, with samples that fit the file.
func sampleSizes(table []byte, limit uint64) ([]uint32, bool) {
	sizes, ok := box(table, "stsz")
	if !ok || len(sizes) < stszHeaderSize {
		return nil, false
	}

	every := binary.BigEndian.Uint32(sizes[fullBoxSize:])
	count := binary.BigEndian.Uint32(sizes[fullBoxSize+countSize:])
	listed := sizes[stszHeaderSize:]

	if every == 0 && uint64(count) > uint64(len(listed)/countSize) || uint64(count)*uint64(every) > limit {
		return nil, false
	}

	out := make([]uint32, 0, count)

	for at := range int(count) {
		size := every
		if every == 0 {
			size = binary.BigEndian.Uint32(listed[countSize*at:])
		}

		out = append(out, size)
	}

	return out, true
}

// chunkOffsets reads where each chunk starts from a sample table's stco or co64 box.
//
// Parameters:
//   - table: the sample table's body.
//
// Returns:
//   - []uint64: each chunk's offset in the file.
//   - bool: whether a box was found whole.
func chunkOffsets(table []byte) ([]uint64, bool) {
	width := offsetBytes

	offsets, ok := box(table, "stco")
	if !ok {
		offsets, ok = box(table, "co64")
		width = largeOffsetBytes
	}

	if !ok || len(offsets) < fullBoxSize+countSize {
		return nil, false
	}

	count := int(binary.BigEndian.Uint32(offsets[fullBoxSize:]))
	listed := offsets[fullBoxSize+countSize:]

	if count > len(listed)/width {
		return nil, false
	}

	out := make([]uint64, 0, count)

	for at := range count {
		entry := listed[width*at:]

		if width == offsetBytes {
			out = append(out, uint64(binary.BigEndian.Uint32(entry)))
		} else {
			out = append(out, binary.BigEndian.Uint64(entry))
		}
	}

	return out, true
}

// sampleRuns reads the entries of a sample table's stsc box, which say how many samples each run
// of chunks holds.
//
// Parameters:
//   - table: the sample table's body.
//
// Returns:
//   - []byte: the entries.
//   - bool: whether the box was found whole, with at least one entry.
func sampleRuns(table []byte) ([]byte, bool) {
	runs, ok := box(table, "stsc")
	if !ok || len(runs) < fullBoxSize+countSize {
		return nil, false
	}

	count := int(binary.BigEndian.Uint32(runs[fullBoxSize:]))
	entries := runs[fullBoxSize+countSize:]

	if count < 1 || count > len(entries)/stscEntrySize {
		return nil, false
	}

	return entries[:stscEntrySize*count], true
}

// samplesInChunk returns how many samples a chunk holds, by the stsc run it falls in.
//
// Parameters:
//   - runs: the stsc entries.
//   - chunk: the chunk's number, from 1.
//
// Returns:
//   - int: the number of samples.
func samplesInChunk(runs []byte, chunk int) int {
	samples := 0

	for at := 0; at < len(runs); at += stscEntrySize {
		entry := runs[at:]
		if int(binary.BigEndian.Uint32(entry)) > chunk {
			break
		}

		samples = int(binary.BigEndian.Uint32(entry[countSize:]))
	}

	return samples
}
