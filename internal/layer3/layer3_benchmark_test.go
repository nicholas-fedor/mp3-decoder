// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// BenchmarkMainData measures handing out the main data of each frame of a 128 kbit/s joint stereo
// file that LAME encoded.
func BenchmarkMainData(b *testing.B) {
	data, err := os.ReadFile("testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	var (
		begins []int
		owns   [][]byte
		side   SideInfo
	)

	for at := 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(b, ok)

		end := at + head.Size()
		require.NoError(b, side.Parse(head, data[at:end]))

		begins = append(begins, side.MainDataBegin)
		owns = append(owns, data[at+head.DataAt():end])
		at = end
	}

	var reservoir Reservoir

	for b.Loop() {
		reservoir.Reset()

		for i, own := range owns {
			reservoir.MainData(begins[i], own)
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(owns)), "ns/frame")
}
