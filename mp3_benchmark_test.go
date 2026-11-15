// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
)

// BenchmarkRead measures opening and decoding a joint stereo file that LAME encoded at -V 2, 0.5 s
// of a tone with clicks, and reports how many times faster than real time it decodes.
func BenchmarkRead(b *testing.B) {
	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	buffer := make([][2]float64, 4096)
	samples := 0

	b.ReportAllocs()

	for b.Loop() {
		decoder, err := mp3.OpenStream(b.Context(), bytes.NewReader(data))
		require.NoError(b, err)

		for {
			read, err := decoder.Read(buffer)
			samples += read

			if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
				break
			}
		}
	}

	b.ReportMetric(float64(samples)/44100/b.Elapsed().Seconds(), "x-realtime")
}

// BenchmarkSeek measures seeking a file opened from memory to samples spread over it, then reading
// the frame at each. The file is the joint stereo fixture LAME encoded at -V 2, joined to three
// copies of itself, which the walk in Open indexes.
func BenchmarkSeek(b *testing.B) {
	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	decoder, err := mp3.Open(b.Context(), bytes.NewReader(bytes.Repeat(data, 4)))
	require.NoError(b, err)

	length, _ := decoder.Len()
	buffer := make([][2]float64, 1152)
	sample := int64(0)

	b.ReportAllocs()

	for b.Loop() {
		sample = (sample + 30011) % (length - int64(len(buffer)))

		err = decoder.Seek(b.Context(), sample)
		if err == nil {
			_, err = decoder.Read(buffer)
		}
	}

	require.NoError(b, err)
}
