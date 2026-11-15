// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
)

// TestReferences decodes each MPEG-1 fixture and checks it against the references mpg123 and
// ffmpeg decoded, channel by channel.
//
// Every sample must be there, and the difference must stay within full accuracy. It must also
// stay within twice the largest RMS difference measured from each tool, so a loss of precision
// shows long before it breaks the bound.
func TestReferences(t *testing.T) {
	t.Parallel()

	ceilings := map[string]float64{"mpg123": 1.6e-7, "ffmpeg": 3e-7}

	for _, name := range []string{"mpeg1-joint", "mpeg1-mono", "protected"} {
		got := decodeFile(t, name+".mp3")

		for tool, ceiling := range ceilings {
			t.Run(name+" "+tool, func(t *testing.T) {
				t.Parallel()

				wave := readReference(t, name+"."+tool+".f32", len(got))
				ref := conformance.Reference{
					Name: name, MD5: "", Channels: []int{0, 1}[:len(wave.Channels)], Start: 0, Alternative: false,
				}

				differences, err := ref.Measure(got, wave)
				require.NoError(t, err)
				require.Len(t, wave.Channels[0], len(got), "samples")

				for ch, difference := range differences {
					assert.True(t, difference.Full(), "channel %d: %+v", ch, difference)
					assert.Less(t, difference.RMS, ceiling, "channel %d RMS", ch)
				}
			})
		}
	}
}

// TestReaders checks that the samples do not depend on how the source hands out its bytes or how
// many samples each Read asks for.
func TestReaders(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(t, err)

	want := decodeFile(t, "mpeg1-joint.mp3")

	sources := map[string]func() io.Reader{
		"one byte at a time": func() io.Reader { return iotest.OneByteReader(bytes.NewReader(data)) },
		"half reads":         func() io.Reader { return iotest.HalfReader(bytes.NewReader(data)) },
		"data with io.EOF":   func() io.Reader { return iotest.DataErrReader(bytes.NewReader(data)) },
	}

	for name, open := range sources {
		for _, size := range []int{1, 7, 1152, 5000} {
			decoder, err := mp3.OpenStream(t.Context(), open())
			require.NoError(t, err, name)

			got := readAll(t, decoder, size)
			assert.Equal(t, want, got, "%s, %d samples a read", name, size)
		}
	}
}

// TestReadEnd checks that a Read for no samples returns none, and that every Read after the last
// sample returns io.EOF itself.
func TestReadEnd(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/mpeg1-mono.mp3")
	require.NoError(t, err)

	defer file.Close()

	decoder, err := mp3.OpenStream(t.Context(), file)
	require.NoError(t, err)

	read, err := decoder.Read(nil)
	assert.Zero(t, read)
	require.NoError(t, err)

	readAll(t, decoder, 4096)

	for range 2 {
		read, err = decoder.Read(make([][2]float64, 10))
		assert.Zero(t, read)
		assert.Equal(t, io.EOF, err)
	}
}

// decodeFile decodes a fixture with reads of 1000 samples.
func decodeFile(t *testing.T, name string) [][2]float64 {
	t.Helper()

	file, err := os.Open("testdata/" + name)
	require.NoError(t, err)

	defer file.Close()

	decoder, err := mp3.OpenStream(t.Context(), file)
	require.NoError(t, err)

	return readAll(t, decoder, 1000)
}

// readAll reads a decoder to its end, asking for a count of samples each time.
func readAll(t *testing.T, decoder *mp3.Decoder, size int) [][2]float64 {
	t.Helper()

	var out [][2]float64

	buffer := make([][2]float64, size)

	for {
		read, err := decoder.Read(buffer)
		out = append(out, buffer[:read]...)

		if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
			return out
		}

		require.NoError(t, err)
		require.Positive(t, read)
	}
}

// readReference reads a reference of interleaved 32-bit little-endian floats, with as many
// channels as fit a stream of a number of samples.
func readReference(t *testing.T, name string, samples int) conformance.Waveform {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	values := len(data) / 4
	channels := max(values/max(samples, 1), 1)
	wave := conformance.Waveform{Channels: make([][]float64, channels), Rate: 0}

	for at := range values {
		value := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4*at:])))

		wave.Channels[at%channels] = append(wave.Channels[at%channels], value)
	}

	return wave
}
