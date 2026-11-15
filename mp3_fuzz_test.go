// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"context"
	"io"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// FuzzRead opens and decodes any bytes as a stream, and checks that decoding ends, that every
// sample is a number, and that no more samples come out than frames fit the bytes.
//
// The seed corpus holds the first frames of each MPEG-1 Layer III fixture and of the Layer II
// fixtures in joint stereo and of the lower sampling frequencies, which the fuzzer damages.
func FuzzRead(f *testing.F) {
	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3", "layer2-joint.mp2", "layer2-mpeg2.mp2"} {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(f, err)

		f.Add(data[:min(len(data), 4000)])
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		decoder, err := mp3.OpenStream(t.Context(), bytes.NewReader(data))
		if err != nil {
			return
		}

		buffer := make([][2]float64, 1000)
		total := 0

		for range data {
			read, err := decoder.Read(buffer)
			total += read

			for _, pair := range buffer[:read] {
				require.False(t, math.IsNaN(pair[0]) || math.IsNaN(pair[1]))
			}

			// Every frame takes at least its header's bytes.
			require.LessOrEqual(t, total, (len(data)/frame.HeaderSize)*frame.MaxSamples)

			if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
				return
			}
		}

		t.Fatal("decoding did not end")
	})
}

// FuzzCorrupt damages up to 16 consecutive bytes of a fixture and checks that decoding carries on
// past the damage to the end of the stream.
//
// It costs only the frames it touches and those a damaged header's size jumps over, never the
// rest of the stream. The stream holds the fixture three times over and only the first copy is
// damaged, so a stream that ends early loses at least the 42 frames of the other two. The seed
// corpus damages a header, side information and main data.
func FuzzCorrupt(f *testing.F) {
	const most = 16

	fixture, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(f, err)

	data := slices.Concat(fixture, fixture, fixture)

	want, err := decodeAll(f.Context(), data)
	require.NoError(f, err)

	// The frame at byte 4976 has its side information from byte 4980 and main data after it.
	f.Add(uint16(4976), []byte{0xff, 0xff, 0xff, 0xff})
	f.Add(uint16(4990), []byte{0xff, 0x10, 0x20})
	f.Add(uint16(5100), []byte{0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55})

	f.Fuzz(func(t *testing.T, at uint16, damage []byte) {
		stream := slices.Clone(data)
		for offset, value := range damage[:min(len(damage), most)] {
			stream[(int(at)+offset)%len(fixture)] ^= value
		}

		got, err := decodeAll(t.Context(), stream)
		require.NoError(t, err)

		// A damaged header of the stream gives a frame of at most 1045 bytes, at 320 kbit/s with
		// padding. The fixture's frames hold at least 104 bytes, so it passes over at most 10 of
		// them besides its own.
		assert.GreaterOrEqual(t, len(got), len(want)-11*frame.MaxSamples)
	})
}

// decodeAll opens and decodes a stream held in memory to its end, passing over frames that fail
// to decode.
func decodeAll(ctx context.Context, data []byte) ([][2]float64, error) {
	decoder, err := mp3.OpenStream(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var out [][2]float64

	buffer := make([][2]float64, 1000)

	for {
		read, err := decoder.Read(buffer)
		out = append(out, buffer[:read]...)

		if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
			return out, nil
		}
	}
}

// FuzzJoin opens a fixture at any byte, as on joining a radio stream, and checks that the samples
// soon match the end of the whole decode.
//
// The fuzzer picks the fixture and the cut. The seed corpus cuts each fixture inside its first
// frame and partway through.
func FuzzJoin(f *testing.F) {
	type fixture struct {
		data   []byte
		starts []int
		want   [][2]float64
		end    int
	}

	names := []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3", "mpeg2.mp3", "mpeg25.mp3", "free.mp3"}
	fixtures := make([]fixture, 0, len(names))

	for _, name := range names {
		data, starts := audioFrames(f, name)

		// A first frame needs the header after it to confirm it, and a free-format one needs two.
		last := len(starts) - 2
		if head, _ := frame.Parse(data[starts[0]:]); head.Free {
			last--
		}

		fixtures = append(fixtures, fixture{data: data, starts: starts, want: decodeBytes(f, data), end: starts[last]})
	}

	for pick := range fixtures {
		f.Add(uint8(pick), uint16(1))
		f.Add(uint8(pick), uint16(fixtures[pick].end/2))
	}

	f.Fuzz(func(t *testing.T, pick uint8, at uint16) {
		chosen := fixtures[int(pick)%len(fixtures)]

		checkJoin(t, chosen.data, chosen.starts, chosen.want, 1+int(at)%(chosen.end-1))
	})
}

// FuzzSeek seeks a fixture opened as a file to any samples, and checks that the samples after each
// seek are those of a decode from the start, bit for bit.
//
// Each fixture is taken alone and joined to a copy of itself, which a walk indexes. The fuzzer
// picks the fixture, then for each seek two bytes for the sample and one for the samples read
// after it. The seed corpus seeks each fixture to its end, its start and its middle.
func FuzzSeek(f *testing.F) {
	type fixture struct {
		data []byte
		want [][2]float64
	}

	names := []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3", "mpeg2.mp3", "mpeg25.mp3", "free.mp3"}
	fixtures := make([]fixture, 0, 2*len(names))

	for _, name := range names {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(f, err)

		for _, stream := range [][]byte{data, slices.Concat(data, data)} {
			fixtures = append(fixtures, fixture{data: stream, want: readAll(f, openFile(f, stream), 1000)})
		}
	}

	for pick := range fixtures {
		f.Add(uint8(pick), []byte{0xff, 0xff, 10, 0, 0, 200, 0x80, 0, 30})
	}

	f.Fuzz(func(t *testing.T, pick uint8, seeks []byte) {
		chosen := fixtures[int(pick)%len(fixtures)]
		length := int64(len(chosen.want))

		decoder := openFile(t, chosen.data)
		buffer := make([][2]float64, 256*16)

		for ; len(seeks) >= 3; seeks = seeks[3:] {
			target := (int64(seeks[0])<<8 | int64(seeks[1])) * length / 0xffff

			require.NoError(t, decoder.Seek(t.Context(), target))
			require.Equal(t, target, decoder.Position())

			read := readFull(t, decoder, buffer[:min(16*(int64(seeks[2])+1), length-target)])

			if !slices.Equal(chosen.want[target:target+int64(read)], buffer[:read]) {
				require.Failf(t, "the samples differ", "after a seek to sample %d of %d", target, length)
			}
		}
	})
}
