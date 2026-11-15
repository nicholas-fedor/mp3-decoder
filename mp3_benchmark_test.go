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

// BenchmarkRead measures opening and decoding a 128 kbit/s joint stereo file that LAME encoded,
// 0.55 s of audio, and reports how many times faster than real time it decodes.
func BenchmarkRead(b *testing.B) {
	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	buffer := make([][2]float64, 4096)
	samples := 0

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
