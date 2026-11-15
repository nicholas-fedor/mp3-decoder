// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"bytes"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
)

// FuzzRead opens and decodes any bytes as a stream, and checks that decoding ends and that every
// sample is a number.
//
// The seed corpus holds the first frames of each MPEG-1 fixture, which the fuzzer damages.
func FuzzRead(f *testing.F) {
	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "protected.mp3"} {
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

		for range data {
			read, err := decoder.Read(buffer)

			for _, pair := range buffer[:read] {
				require.False(t, math.IsNaN(pair[0]) || math.IsNaN(pair[1]))
			}

			if err == io.EOF { //nolint:errorlint // Read returns io.EOF itself.
				return
			}
		}

		t.Fatal("decoding did not end")
	})
}
