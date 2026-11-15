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
)

// Full accuracy, as ISO/IEC 11172-4 clause 2.6.3 bounds a decoder's difference from a reference
// for output between -1 and 1.
var (
	// fullAccuracyRMS bounds the root mean square difference, 2^-15/√12.
	fullAccuracyRMS = math.Ldexp(1, -15) / math.Sqrt(12)
	// fullAccuracyMax bounds the largest difference, 2^-14.
	fullAccuracyMax = math.Ldexp(1, -14)
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

				want := readReference(t, name+"."+tool+".f32")
				channels := len(want) / len(got)
				require.Equal(t, len(want), len(got)*channels, "samples")

				for ch := range channels {
					var squares, largest float64

					for at, pair := range got {
						difference := pair[ch] - want[at*channels+ch]
						squares += difference * difference
						largest = max(largest, math.Abs(difference))
					}

					rms := math.Sqrt(squares / float64(len(got)))

					assert.Less(t, rms, fullAccuracyRMS, "channel %d RMS", ch)
					assert.LessOrEqual(t, largest, fullAccuracyMax, "channel %d largest", ch)
					assert.Less(t, rms, ceiling, "channel %d RMS", ch)
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

// readReference reads a reference of 32-bit little-endian floats.
func readReference(t *testing.T, name string) []float64 {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	samples := make([]float64, len(data)/4)
	for at := range samples {
		samples[at] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4*at:])))
	}

	return samples
}
