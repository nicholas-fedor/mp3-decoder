// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"encoding/binary"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mp4Layout shapes a test MP4 file.
type mp4Layout struct {
	// samples holds each sample's bytes.
	samples [][]byte
	// runs holds stsc entries, each a first chunk from 1 and its samples per chunk.
	runs [][2]uint32
	// chunks is the number of chunks.
	chunks int
	// uniform stores one size for every sample, which must then all be as long.
	uniform bool
	// wide stores the chunk offsets as 64 bits, in co64.
	wide bool
	// largeMoov gives the moov box a 64-bit size.
	largeMoov bool
}

// TestMP4Frames checks that the samples come out end to end with sizes listed or uniform, chunks
// of different runs, 32-bit and 64-bit offsets, and a moov box with a 64-bit size, and that
// malformed files fail.
func TestMP4Frames(t *testing.T) {
	t.Parallel()

	samples := [][]byte{[]byte("one"), []byte("two"), []byte("three!"), []byte("four"), []byte("five")}
	even := [][]byte{[]byte("aaaa"), []byte("bbbb"), []byte("cccc")}

	valid := []struct {
		name   string
		layout mp4Layout
		want   []byte
	}{
		{"listed sizes in one chunk", mp4Layout{samples: samples, runs: [][2]uint32{{1, 5}}, chunks: 1}, []byte("onetwothree!fourfive")},
		{"runs of chunks", mp4Layout{samples: samples, runs: [][2]uint32{{1, 2}, {3, 1}}, chunks: 3}, []byte("onetwothree!fourfive")},
		{"uniform sizes", mp4Layout{samples: even, runs: [][2]uint32{{1, 1}}, chunks: 3, uniform: true}, []byte("aaaabbbbcccc")},
		{"64-bit offsets", mp4Layout{samples: even, runs: [][2]uint32{{1, 3}}, chunks: 1, wide: true}, []byte("aaaabbbbcccc")},
		{"64-bit box size", mp4Layout{samples: even, runs: [][2]uint32{{1, 3}}, chunks: 1, largeMoov: true}, []byte("aaaabbbbcccc")},
	}

	for _, tt := range valid {
		got, err := mp4Frames(mp4File(tt.layout))
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}

	file := mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 5}}, chunks: 1})

	invalid := map[string][]byte{
		"no moov":                   []byte("\x00\x00\x00\x08free"),
		"cut short":                 file[:len(file)/2],
		"a box past the end":        slices.Concat([]byte{0x7f, 0, 0, 0}, []byte("moov")),
		"too few samples in chunks": mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 2}}, chunks: 2}),
		"too many samples":          mp4File(mp4Layout{samples: samples, runs: [][2]uint32{{1, 4}}, chunks: 2}),
		"no stsc entries":           mp4File(mp4Layout{samples: samples, chunks: 1}),
	}

	// sizes gives one sample of 1 byte.
	sizes := mp4Box("stsz", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1})
	runs := mp4Box("stsc", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1})
	offsets := mp4Box("stco", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0})

	invalid["no stsz"] = withTable(slices.Concat(runs, offsets))
	invalid["stsz cut short"] = withTable(slices.Concat(mp4Box("stsz", []byte{0, 0}), runs, offsets))
	invalid["fewer sizes than samples"] = withTable(slices.Concat(mp4Box("stsz", []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5}), runs, offsets))
	invalid["uniform sizes past the file"] = withTable(slices.Concat(mp4Box("stsz", []byte{0, 0, 0, 0, 0, 0, 3, 0xe8, 0, 0, 3, 0xe8}), runs, offsets))
	invalid["no chunk offsets"] = withTable(slices.Concat(sizes, runs))
	invalid["stco cut short"] = withTable(slices.Concat(sizes, runs, mp4Box("stco", []byte{0, 0})))
	invalid["fewer offsets than chunks"] = withTable(slices.Concat(sizes, runs, mp4Box("stco", []byte{0, 0, 0, 0, 0, 0, 0, 9})))
	invalid["no stsc"] = withTable(slices.Concat(sizes, offsets))
	invalid["a box to the end before moov"] = slices.Concat([]byte{0, 0, 0, 0}, []byte("free"), mp4Box("moov", nil))
	invalid["a 64-bit size cut short"] = slices.Concat([]byte{0, 0, 0, largeSize}, []byte("moov"))

	for name, data := range invalid {
		_, err := mp4Frames(data)
		require.ErrorIs(t, err, ErrMP4, name)
	}
}

// withTable builds an MP4 file whose first track's sample table holds the given boxes.
func withTable(table []byte) []byte {
	return mp4Box("moov", mp4Box("trak", mp4Box("mdia", mp4Box("minf", mp4Box("stbl", table)))))
}

// mp4File builds an MP4 file whose first track holds samples in chunks in its mdat box.
//
// It builds the file twice, the second time with the chunk offsets the first build places.
func mp4File(layout mp4Layout) []byte {
	var payload []byte

	for _, sample := range layout.samples {
		payload = append(payload, sample...)
	}

	first := mp4Build(layout, payload, 0)

	return mp4Build(layout, payload, len(first)-len(payload))
}

// mp4Build builds an MP4 file whose samples start at a given offset.
func mp4Build(layout mp4Layout, payload []byte, dataAt int) []byte {
	every := uint32(0)
	if layout.uniform {
		every = uint32(len(layout.samples[0]))
	}

	sizes := binary.BigEndian.AppendUint32(make([]byte, fullBoxSize), every)
	sizes = binary.BigEndian.AppendUint32(sizes, uint32(len(layout.samples)))

	if !layout.uniform {
		for _, sample := range layout.samples {
			sizes = binary.BigEndian.AppendUint32(sizes, uint32(len(sample)))
		}
	}

	runs := binary.BigEndian.AppendUint32(make([]byte, fullBoxSize), uint32(len(layout.runs)))
	for _, run := range layout.runs {
		runs = binary.BigEndian.AppendUint32(binary.BigEndian.AppendUint32(runs, run[0]), run[1])
		runs = binary.BigEndian.AppendUint32(runs, 1)
	}

	offsets := binary.BigEndian.AppendUint32(make([]byte, fullBoxSize), uint32(layout.chunks))
	at := dataAt

	for chunk := range layout.chunks {
		if layout.wide {
			offsets = binary.BigEndian.AppendUint64(offsets, uint64(at))
		} else {
			offsets = binary.BigEndian.AppendUint32(offsets, uint32(at))
		}

		at += chunkBytes(layout, chunk+1)
	}

	offsetBox := "stco"
	if layout.wide {
		offsetBox = "co64"
	}

	table := slices.Concat(mp4Box("stsd", nil), mp4Box("stsz", sizes), mp4Box("stsc", runs), mp4Box(offsetBox, offsets))
	track := mp4Box("trak", mp4Box("mdia", mp4Box("minf", mp4Box("stbl", table))))

	moov := mp4Box("moov", track)
	if layout.largeMoov {
		moov = slices.Concat(binary.BigEndian.AppendUint32(nil, largeSize), []byte("moov"),
			binary.BigEndian.AppendUint64(nil, uint64(largeHeaderSize+len(track))), track)
	}

	return slices.Concat(mp4Box("ftyp", []byte("isom")), moov, mp4Box("mdat", payload))
}

// chunkBytes returns the bytes of the samples a chunk holds, by the layout's runs.
func chunkBytes(layout mp4Layout, chunk int) int {
	first := 0

	for prior := 1; prior < chunk; prior++ {
		first += samplesOf(layout, prior)
	}

	last := min(first+samplesOf(layout, chunk), len(layout.samples))
	size := 0

	for _, sample := range layout.samples[min(first, len(layout.samples)):last] {
		size += len(sample)
	}

	return size
}

// samplesOf returns how many samples the layout's runs give a chunk.
func samplesOf(layout mp4Layout, chunk int) int {
	samples := 0

	for _, run := range layout.runs {
		if int(run[0]) <= chunk {
			samples = int(run[1])
		}
	}

	return samples
}

// mp4Box builds a box of a type around a body.
func mp4Box(kind string, body []byte) []byte {
	return slices.Concat(binary.BigEndian.AppendUint32(nil, uint32(boxHeaderSize+len(body))), []byte(kind), body)
}
