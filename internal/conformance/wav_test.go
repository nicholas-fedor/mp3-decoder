// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wavFormat describes the fmt chunk a test WAV file carries.
type wavFormat struct {
	// guid is the sub-format GUID of an extensible chunk.
	guid []byte
	// tag is the format tag.
	tag uint16
	// channels is the number of channels.
	channels uint16
	// align is the size of a sample frame.
	align uint16
	// bits is the sample width.
	bits uint16
	// size is the chunk's size, 16 for PCM and 40 for extensible.
	size int
}

// pcm24 returns the fmt chunk of 24-bit integer PCM.
func pcm24(channels uint16) wavFormat {
	return wavFormat{guid: nil, tag: formatPCM, channels: channels, align: 3 * channels, bits: 24, size: fmtSize}
}

// extensible24 returns the fmt chunk of 24-bit integer PCM as WAVE_FORMAT_EXTENSIBLE.
func extensible24(channels uint16) wavFormat {
	return wavFormat{
		guid: pcmGUID, tag: formatExtensible, channels: channels, align: 3 * channels, bits: 24, size: extensibleSize,
	}
}

// chunk returns a chunk with its header, padded to an even size.
func chunk(id string, body []byte) []byte {
	data := binary.LittleEndian.AppendUint32([]byte(id), uint32(len(body)))
	data = append(data, body...)

	if len(body)%2 == 1 {
		data = append(data, 0)
	}

	return data
}

// fmtChunk returns the fmt chunk of a format at 44100 Hz.
func fmtChunk(format wavFormat) []byte {
	body := make([]byte, format.size)

	binary.LittleEndian.PutUint16(body, format.tag)
	binary.LittleEndian.PutUint16(body[2:], format.channels)
	binary.LittleEndian.PutUint32(body[4:], 44100)
	binary.LittleEndian.PutUint16(body[12:], format.align)
	binary.LittleEndian.PutUint16(body[14:], format.bits)

	if format.size >= extensibleSize {
		copy(body[subFormatAt:], format.guid)
	}

	return chunk("fmt ", body)
}

// riff wraps chunks in a RIFF WAVE header.
func riff(chunks ...[]byte) []byte {
	var body []byte

	for _, piece := range chunks {
		body = append(body, piece...)
	}

	data := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(4+len(body)))

	return append(append(data, "WAVE"...), body...)
}

// TestParseWAV reads plain and extensible files, with chunks before and between fmt and data, and
// checks the extreme sample values and a partial sample frame at the end.
func TestParseWAV(t *testing.T) {
	t.Parallel()

	// Samples: -1, the largest, 1/2^23 and -1/2^23, then a partial frame.
	data := []byte{0x00, 0x00, 0x80, 0xff, 0xff, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xff, 0x12}
	want := [][]float64{{-1, 1.0 / fullScale}, {float64(fullScale-1) / fullScale, -1.0 / fullScale}}

	tests := []struct {
		name string
		file []byte
	}{
		{"pcm", riff(fmtChunk(pcm24(2)), chunk("data", data))},
		{"extensible", riff(fmtChunk(extensible24(2)), chunk("data", data))},
		{"other chunks", riff(chunk("JUNK", []byte{1, 2, 3}), fmtChunk(pcm24(2)), chunk("afsp", nil), chunk("data", data))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			wave, err := ParseWAV(test.file)
			require.NoError(t, err)
			assert.Equal(t, 44100, wave.Rate)
			assert.Equal(t, want, wave.Channels)
		})
	}
}

// TestParseWAVErrors checks that every file other than 24-bit integer PCM, and every damaged file,
// is refused with ErrWAV.
func TestParseWAVErrors(t *testing.T) {
	t.Parallel()

	data := chunk("data", make([]byte, 6))
	otherGUID := extensible24(2)
	otherGUID.guid = []byte{3}

	tests := []struct {
		name string
		file []byte
	}{
		{"empty", nil},
		{"not riff", append([]byte("RIFX\x00\x00\x00\x00WAVE"), data...)},
		{"not wave", append([]byte("RIFF\x00\x00\x00\x00AVI "), data...)},
		{"no chunks", riff()},
		{"no data", riff(fmtChunk(pcm24(2)))},
		{"data first", riff(data, fmtChunk(pcm24(2)))},
		{"chunk past the end", riff(fmtChunk(pcm24(2)), []byte("data\xff\x00\x00\x00"))},
		{"short fmt", riff(chunk("fmt ", make([]byte, 14)), data)},
		{"float", riff(fmtChunk(wavFormat{nil, 3, 2, 6, 24, fmtSize}), data)},
		{"short extensible", riff(fmtChunk(wavFormat{nil, formatExtensible, 2, 6, 24, fmtSize}), data)},
		{"extensible float", riff(fmtChunk(otherGUID), data)},
		{"16 bits", riff(fmtChunk(wavFormat{nil, formatPCM, 2, 4, 16, fmtSize}), data)},
		{"no channels", riff(fmtChunk(wavFormat{nil, formatPCM, 0, 0, 24, fmtSize}), data)},
		{"padded samples", riff(fmtChunk(wavFormat{nil, formatPCM, 2, 8, 24, fmtSize}), data)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseWAV(test.file)
			require.ErrorIs(t, err, ErrWAV)
		})
	}
}
