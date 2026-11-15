// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// TestSeek checks that the samples after a seek are those of a decode from the start, bit for bit,
// on every fixture and on a Layer I stream.
//
// Each fixture is opened alone, whose tag frame gives its length so no walk indexes it, and joined
// to a copy of itself, which a walk indexes. The seeks run far forward first, then back and forth,
// to the first and last samples, to frame edges, and to random samples.
func TestSeek(t *testing.T) {
	t.Parallel()

	streams := map[string][]byte{"Layer I": layer1Stream(t, 30)}

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3", "free.mp3", "mpeg2.mp3", "mpeg25.mp3"} {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(t, err)

		streams[name] = data
	}

	for name, data := range streams {
		for kind, stream := range map[string][]byte{"alone": data, "joined to a copy": slices.Concat(data, data)} {
			t.Run(name+" "+kind, func(t *testing.T) {
				t.Parallel()

				whole := openFile(t, stream)
				want := readAll(t, whole, 1000)
				length := int64(len(want))

				decoder := openFile(t, stream)
				_, head, _ := frame.Find(stream, nil)
				samples := int64(head.Samples())

				targets := make([]int64, 0, 32)
				targets = append(targets, length*3/4, 0, length-1, 1, samples-1, samples, length/2, 2*samples+7)
				random := rand.New(rand.NewPCG(1, 2))

				for len(targets) < cap(targets) {
					targets = append(targets, random.Int64N(length))
				}

				buffer := make([][2]float64, 3000)

				for _, target := range targets {
					require.NoError(t, decoder.Seek(t.Context(), target))
					require.Equal(t, target, decoder.Position())

					read := readFull(t, decoder, buffer[:min(int64(len(buffer)), length-target)])

					if !slices.Equal(want[target:target+int64(read)], buffer[:read]) {
						require.Failf(t, "the samples differ", "after a seek to sample %d of %d", target, length)
					}

					assert.Equal(t, target+int64(read), decoder.Position())
				}

				require.NoError(t, decoder.Seek(t.Context(), length))
				assert.Equal(t, length, decoder.Position())

				read, err := decoder.Read(buffer)
				assert.Zero(t, read)
				require.ErrorIs(t, err, io.EOF)
			})
		}
	}
}

// TestSeekRange checks that a seek outside the stream fails and leaves the decoder where it was,
// and that a stream that cannot seek refuses.
func TestSeekRange(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(t, err)

	decoder := openFile(t, data)
	length, _ := decoder.Len()

	for _, sample := range []int64{-1, length + 1} {
		require.ErrorIs(t, decoder.Seek(t.Context(), sample), mp3.ErrOutOfRange)
		assert.Zero(t, decoder.Position())
	}

	stream, err := mp3.OpenStream(t.Context(), bytes.NewReader(data))
	require.NoError(t, err)
	require.ErrorIs(t, stream.Seek(t.Context(), 0), mp3.ErrNotSeekable)
}

// openFile opens a stream as a file.
func openFile(tb testing.TB, data []byte) *mp3.Decoder {
	tb.Helper()

	decoder, err := mp3.Open(tb.Context(), bytes.NewReader(data))
	require.NoError(tb, err)

	return decoder
}

// readFull reads samples from a decoder until the buffer is full or the stream ends.
func readFull(tb testing.TB, decoder *mp3.Decoder, buffer [][2]float64) int {
	tb.Helper()

	filled := 0

	for filled < len(buffer) {
		read, err := decoder.Read(buffer[filled:])
		filled += read

		if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
			break
		}

		require.NoError(tb, err)
	}

	return filled
}
