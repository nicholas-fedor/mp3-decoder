// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"slices"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestReferences decodes each MPEG-1 fixture without its tag frame and checks it against the
// references mpg123 and ffmpeg decoded, channel by channel.
//
// Neither tool leaves samples out of the references, and without the tag frame the decoder leaves
// out none either.
// Every sample must be there, and the difference must stay within full accuracy. It must also
// stay within twice the largest RMS difference measured from each tool, so a loss of precision
// shows long before it breaks the bound. ffmpeg refuses the free-format fixture, so only mpg123's
// reference checks it.
func TestReferences(t *testing.T) {
	t.Parallel()

	ceilings := map[string]float64{"mpg123": 1.6e-7, "ffmpeg": 3e-7}
	fixtures := map[string][]string{
		"mpeg1-joint": {"mpg123", "ffmpeg"}, "mpeg1-mono": {"mpg123", "ffmpeg"}, "protected": {"mpg123", "ffmpeg"},
		"free": {"mpg123"},
	}

	for name, tools := range fixtures {
		audio, _ := audioFrames(t, name+".mp3")
		got := decodeBytes(t, audio)

		for _, tool := range tools {
			ceiling := ceilings[tool]

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

// TestGapless checks that each fixture with a LAME tag decodes to exactly the 0.5 s of samples
// its encoder took in, and the fixture without one to every sample.
//
// The samples are those of every frame from the encoder delay and the 529 samples of decoder delay
// on, up to the padding less those 529 samples before the end of the frames the tag counts. That
// is what mpg123 leaves out, and its gapless output of each fixture is exactly those samples.
func TestGapless(t *testing.T) {
	t.Parallel()

	samples := map[string]int{
		"mpeg1-joint.mp3": 22050, "mpeg1-mono.mp3": 22050, "protected.mp3": 22050, "free.mp3": 22050,
		"mpeg2.mp3": 11025, "mpeg25.mp3": 5184,
	}

	for name, want := range samples {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := decodeFile(t, name)
			audio, _ := audioFrames(t, name)
			whole := decodeBytes(t, audio)

			data, err := os.ReadFile("testdata/" + name)
			require.NoError(t, err)

			at, head, ok := frame.Find(data, nil)
			require.True(t, ok)

			from, to := 0, len(whole)
			if info, tagged := container.ParseInfo(head, data[at:]); tagged {
				from = info.Delay + 529
				to = int(info.Frames)*head.Samples() - info.Padding + 529
			}

			require.Len(t, got, want)
			assert.True(t, slices.Equal(whole[from:to], got), "the samples kept")
		})
	}
}

// TestLen checks that each fixture opened as a file knows its length, and that the length is the
// number of samples it decodes to.
//
// Each fixture is opened alone, wrapped in tags at both ends, and joined to a copy of itself,
// whose tag frame's byte count then no longer matches, so a walk counts the frames.
func TestLen(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3", "free.mp3", "mpeg2.mp3", "mpeg25.mp3"} {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(t, err)

		streams := map[string][]byte{
			"alone": data,
			"wrapped in tags": slices.Concat(
				mp3test.ID3v2{Body: 300}.Bytes(), data, mp3test.APE{Items: 40, Header: true}.Bytes(), mp3test.Lyrics3(20),
				mp3test.ID3v1(),
			),
			"joined to a copy": slices.Concat(data, data),
		}

		for kind, stream := range streams {
			t.Run(name+" "+kind, func(t *testing.T) {
				t.Parallel()

				decoder, err := mp3.Open(t.Context(), bytes.NewReader(stream))
				require.NoError(t, err)

				length, known := decoder.Len()
				require.True(t, known)

				got := readAll(t, decoder, 1000)
				assert.Equal(t, int64(len(got)), length)

				if kind != "joined to a copy" {
					assert.True(t, slices.Equal(decodeBytes(t, data), got), "the samples differ from the stream's")
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
func readAll(tb testing.TB, decoder *mp3.Decoder, size int) [][2]float64 {
	tb.Helper()

	var out [][2]float64

	buffer := make([][2]float64, size)

	for {
		read, err := decoder.Read(buffer)
		out = append(out, buffer[:read]...)

		if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
			return out
		}

		require.NoError(tb, err)
		require.Positive(tb, read)
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
