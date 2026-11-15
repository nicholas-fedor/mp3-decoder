// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Waveform is a reference waveform, with samples between -1 and 1.
type Waveform struct {
	// Channels holds each channel's samples.
	//
	// Every channel has the same number of samples.
	Channels [][]float64
	// Rate is the sampling frequency in Hz.
	Rate int
}

// WAV layout.
const (
	// riffHeaderSize is the size of the RIFF header: its ID, its size and the WAVE form type.
	riffHeaderSize = 12
	// chunkHeaderSize is the size of a chunk's ID and size.
	chunkHeaderSize = 8
	// fmtSize is the size of the fields of a PCM fmt chunk.
	fmtSize = 16
	// extensibleSize is the size of the fields of a WAVE_FORMAT_EXTENSIBLE fmt chunk.
	extensibleSize = 40
	// subFormatAt is where the sub-format GUID starts in a WAVE_FORMAT_EXTENSIBLE fmt chunk.
	subFormatAt = 24
	// formatPCM is the format tag of integer PCM.
	formatPCM = 1
	// formatExtensible is the format tag of WAVE_FORMAT_EXTENSIBLE.
	formatExtensible = 0xfffe
	// sampleBytes is the size of a 24-bit sample.
	sampleBytes = 3
	// sampleBits is the width of a sample.
	sampleBits = 24
	// byteBits is the width of a byte.
	byteBits = 8
	// fullScale is the magnitude of a 24-bit sample at -1.
	fullScale = 1 << (sampleBits - 1)
	// wrap is 2^24, by which a negative 24-bit sample read as unsigned exceeds its value.
	wrap = 1 << sampleBits
)

// ErrWAV is returned for data that is not a 24-bit integer PCM WAV file.
var ErrWAV = errors.New("conformance: not a 24-bit PCM WAV file")

// pcmGUID is the sub-format GUID of integer PCM in a WAVE_FORMAT_EXTENSIBLE fmt chunk.
var pcmGUID = []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}

// ParseWAV reads a WAV file of 24-bit integer PCM, the format of ISO's references.
//
// The fmt chunk may be plain PCM or WAVE_FORMAT_EXTENSIBLE with the PCM sub-format. Chunks other
// than fmt and data are skipped.
//
// Parameters:
//   - data: the file.
//
// Returns:
//   - Waveform: the samples.
//   - error: [ErrWAV], wrapped with what is wrong.
func ParseWAV(data []byte) (Waveform, error) {
	if len(data) < riffHeaderSize || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return Waveform{}, fmt.Errorf("%w: no RIFF WAVE header", ErrWAV)
	}

	format, body, err := findChunks(data[riffHeaderSize:])
	if err != nil {
		return Waveform{}, err
	}

	channels, rate, err := parseFormat(format)
	if err != nil {
		return Waveform{}, err
	}

	return Waveform{Channels: samples(body, channels), Rate: rate}, nil
}

// findChunks finds the fmt chunk and the data chunk after it.
//
// Parameters:
//   - chunks: the chunks after the RIFF header.
//
// Returns:
//   - format: the fmt chunk's fields.
//   - body: the data chunk's bytes.
//   - err: [ErrWAV] when a chunk runs past the file, or either chunk is missing or out of order.
//
//nolint:nonamedreturns // Same-type returns need names.
func findChunks(chunks []byte) (format, body []byte, err error) {
	for len(chunks) >= chunkHeaderSize {
		id, size := string(chunks[:4]), int(binary.LittleEndian.Uint32(chunks[4:]))
		if size > len(chunks)-chunkHeaderSize {
			return nil, nil, fmt.Errorf("%w: chunk %q runs past the file", ErrWAV, id)
		}

		switch id {
		case "fmt ":
			format = chunks[chunkHeaderSize : chunkHeaderSize+size]
		case "data":
			if format == nil {
				return nil, nil, fmt.Errorf("%w: data before fmt", ErrWAV)
			}

			return format, chunks[chunkHeaderSize : chunkHeaderSize+size], nil
		default:
		}

		chunks = chunks[min(len(chunks), chunkHeaderSize+size+size%2):]
	}

	return nil, nil, fmt.Errorf("%w: no data chunk", ErrWAV)
}

// parseFormat reads a fmt chunk.
//
// Parameters:
//   - format: the chunk's fields.
//
// Returns:
//   - channels: the number of channels, at least 1.
//   - rate: the sampling frequency in Hz.
//   - err: [ErrWAV] for a format other than 24-bit integer PCM.
//
//nolint:nonamedreturns // Same-type returns need names.
func parseFormat(format []byte) (channels, rate int, err error) {
	if len(format) < fmtSize {
		return 0, 0, fmt.Errorf("%w: short fmt chunk", ErrWAV)
	}

	err = checkTag(format)
	if err != nil {
		return 0, 0, err
	}

	channels = int(binary.LittleEndian.Uint16(format[2:]))
	rate = int(binary.LittleEndian.Uint32(format[4:]))

	align := int(binary.LittleEndian.Uint16(format[12:]))
	bits := int(binary.LittleEndian.Uint16(format[14:]))

	if bits != sampleBits || channels == 0 || align != channels*sampleBytes {
		return 0, 0, fmt.Errorf("%w: %d channels of %d bits in %d bytes", ErrWAV, channels, bits, align)
	}

	return channels, rate, nil
}

// checkTag checks that a fmt chunk's format is integer PCM.
//
// Parameters:
//   - format: the chunk's fields, at least those of a PCM fmt chunk.
//
// Returns:
//   - error: [ErrWAV] for another format.
func checkTag(format []byte) error {
	tag := binary.LittleEndian.Uint16(format)

	switch {
	case tag == formatPCM:
		return nil
	case tag != formatExtensible:
		return fmt.Errorf("%w: format tag %#x", ErrWAV, tag)
	case len(format) < extensibleSize:
		return fmt.Errorf("%w: short extensible fmt chunk", ErrWAV)
	case !bytes.Equal(format[subFormatAt:extensibleSize], pcmGUID):
		return fmt.Errorf("%w: sub-format is not PCM", ErrWAV)
	default:
		return nil
	}
}

// samples splits interleaved 24-bit samples into channels between -1 and 1.
//
// A partial sample frame at the end is left out.
//
// Parameters:
//   - body: the data chunk's bytes.
//   - channels: the number of channels.
//
// Returns:
//   - [][]float64: each channel's samples.
func samples(body []byte, channels int) [][]float64 {
	count := len(body) / (channels * sampleBytes)
	split := make([][]float64, 0, channels)

	for range channels {
		split = append(split, make([]float64, 0, count))
	}

	for at := range count * channels {
		raw := body[at*sampleBytes:]

		value := int(raw[0]) | int(raw[1])<<byteBits | int(raw[2])<<(byteBits+byteBits)
		if value >= fullScale {
			value -= wrap
		}

		split[at%channels] = append(split[at%channels], float64(value)/fullScale)
	}

	return split
}
