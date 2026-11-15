// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
)

// BenchmarkMainData measures handing out the main data of each frame of a joint stereo file that
// LAME encoded at -V 2.
func BenchmarkMainData(b *testing.B) {
	data, err := os.ReadFile("../../testdata/mpeg1-joint.mp3")
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

// BenchmarkSpectrum measures reading the scalefactors and decoding the spectral values of each
// channel's granule of a joint stereo file that LAME encoded at -V 2.
func BenchmarkSpectrum(b *testing.B) {
	data, err := os.ReadFile("../../testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	type framed struct {
		main []byte
		side SideInfo
		head frame.Header
	}

	var (
		frames    []framed
		reservoir Reservoir
	)

	for at := 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(b, ok)

		end := at + head.Size()

		var side SideInfo

		require.NoError(b, side.Parse(head, data[at:end]))

		main, _ := reservoir.MainData(side.MainDataBegin, data[at+head.DataAt():end])
		frames = append(frames, framed{main: slices.Clone(main), side: side, head: head})
		at = end
	}

	var (
		stream   bits.Reader
		factors  [2]Scalefactors
		spectrum Spectrum
	)

	granules := 0

	for b.Loop() {
		for at := range frames {
			f := &frames[at]
			start := 0

			for gr := range 2 {
				for ch := range f.head.Channels() {
					info := &f.side.Granules[gr][ch]

					stream.Reset(f.main, start)
					factors[ch].Read(&stream, f.head, &f.side, gr, ch)
					spectrum.Read(&stream, f.head, info, start+info.Part23Length)

					start += info.Part23Length
					granules++
				}
			}
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(granules), "ns/granule")
}

// BenchmarkRequantize measures requantizing each channel's granule of a joint stereo file that
// LAME encoded at -V 2.
func BenchmarkRequantize(b *testing.B) {
	data, err := os.ReadFile("../../testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	type granule struct {
		spectrum Spectrum
		factors  Scalefactors
		info     GranuleInfo
	}

	var (
		granules  []granule
		reservoir Reservoir
		stream    bits.Reader
		factors   [2]Scalefactors
	)

	for at := 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(b, ok)

		end := at + head.Size()

		var side SideInfo

		require.NoError(b, side.Parse(head, data[at:end]))

		main, _ := reservoir.MainData(side.MainDataBegin, data[at+head.DataAt():end])
		start := 0

		for gr := range 2 {
			for ch := range head.Channels() {
				var g granule

				g.info = side.Granules[gr][ch]

				stream.Reset(main, start)
				factors[ch].Read(&stream, head, &side, gr, ch)
				g.spectrum.Read(&stream, head, &g.info, start+g.info.Part23Length)
				g.factors = factors[ch]
				granules = append(granules, g)

				start += g.info.Part23Length
			}
		}

		at = end
	}

	var lines [granuleLines]float64

	for b.Loop() {
		for at := range granules {
			g := &granules[at]
			g.spectrum.Requantize(&lines, &g.info, &g.factors)
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(granules)), "ns/granule")
}

// BenchmarkStereo measures restoring the left and right lines of each granule of a joint stereo
// file that LAME encoded at -V 2, most of whose granules use mid/side stereo.
func BenchmarkStereo(b *testing.B) {
	data, err := os.ReadFile("../../testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	type granule struct {
		lines   [2][granuleLines]float64
		factors Scalefactors
		info    GranuleInfo
		head    frame.Header
	}

	var (
		granules  []granule
		reservoir Reservoir
		stream    bits.Reader
		factors   [2]Scalefactors
		spectrum  Spectrum
	)

	for at := 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(b, ok)

		end := at + head.Size()

		var side SideInfo

		require.NoError(b, side.Parse(head, data[at:end]))

		main, _ := reservoir.MainData(side.MainDataBegin, data[at+head.DataAt():end])
		start := 0

		for gr := range 2 {
			g := granule{head: head, info: side.Granules[gr][1]}

			for ch := range 2 {
				info := &side.Granules[gr][ch]

				stream.Reset(main, start)
				factors[ch].Read(&stream, head, &side, gr, ch)
				spectrum.Read(&stream, head, info, start+info.Part23Length)
				spectrum.Requantize(&g.lines[ch], info, &factors[ch])

				start += info.Part23Length
			}

			g.factors = factors[1]
			granules = append(granules, g)
		}

		at = end
	}

	for b.Loop() {
		for at := range granules {
			g := &granules[at]
			Stereo(g.head, &g.lines[0], &g.lines[1], &g.info, &g.factors)
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(granules)), "ns/granule")
}

// BenchmarkReorder measures reordering one channel's granule of short blocks at 44.1 kHz.
func BenchmarkReorder(b *testing.B) {
	var lines [granuleLines]float64

	for line := range lines {
		lines[line] = float64(line)
	}

	head := frame.Header{SampleRate: 44100}
	info := GranuleInfo{BlockType: ShortBlocks}

	for b.Loop() {
		Reorder(head, &lines, &info)
	}
}

// BenchmarkDecode measures decoding each frame of a joint stereo file that LAME encoded at -V 2
// through every stage, from its bytes to its PCM samples.
func BenchmarkDecode(b *testing.B) {
	data, err := os.ReadFile("../../testdata/mpeg1-joint.mp3")
	require.NoError(b, err)

	var (
		frames  [][]byte
		headers []frame.Header
	)

	for at := 0; at < len(data); {
		head, ok := frame.Parse(data[at:])
		require.True(b, ok)

		frames = append(frames, data[at:at+head.Size()])
		headers = append(headers, head)
		at += head.Size()
	}

	var decoder Decoder

	pcm := make([][2]float64, frame.MaxSamples)

	b.ReportAllocs()

	for b.Loop() {
		for i, data := range frames {
			_, _, err = decoder.Decode(pcm, headers[i], data)
		}
	}

	require.NoError(b, err)
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(frames)), "ns/frame")
}
