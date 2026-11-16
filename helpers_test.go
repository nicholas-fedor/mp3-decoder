// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3

import (
	"bytes"
	"errors"
	"io"
	"testing/iotest"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// Headers of the streams the tests build.
var (
	// stereoHead is an MPEG-1 Layer III stereo header at 128 kbit/s.
	stereoHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 9}
	// lsfHead is an MPEG-2 Layer III header at 64 kbit/s.
	lsfHead = mp3test.Header{Version: mp3test.MPEG2, Layer: mp3test.Layer3, Bitrate: 8}
	// freeHead is an MPEG-1 Layer III free-format header.
	freeHead = mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3}
)

// errSource is the error the failing sources return.
var errSource = errors.New("source failed")

// frames builds a count of silent frames of a header.
//
// A free-format header takes frames of 417 bytes.
func frames(head mp3test.Header, count int) []byte {
	size := 417

	if parsed, _ := frame.Parse(head.Bytes()); !parsed.Free {
		size = parsed.Size()
	}

	out := make([]byte, 0, size*count)

	for range count {
		out = append(out, head.Frame(size)...)
	}

	return out
}

// failingSeeker is a file whose seeks fail once a count of them has succeeded.
type failingSeeker struct {
	*bytes.Reader

	// seeks is the number of seeks left to succeed.
	seeks int
}

// Seek seeks as the reader does, or fails once the seeks have run out.
func (f *failingSeeker) Seek(offset int64, whence int) (int64, error) {
	if f.seeks == 0 {
		return 0, errSource
	}

	f.seeks--

	return f.Reader.Seek(offset, whence)
}

// failingReader is a file whose reads fail once they have handed out a count of bytes.
type failingReader struct {
	*bytes.Reader

	// reads is the number of bytes left to hand out.
	reads int
}

// Read reads as the reader does, up to the bytes left, and fails once they have run out.
func (f *failingReader) Read(p []byte) (int, error) {
	if f.reads == 0 {
		return 0, errSource
	}

	read, err := f.Reader.Read(p[:min(len(p), f.reads)])
	f.reads -= read

	return read, err
}

// failAfter returns a source that hands out data, then fails.
func failAfter(data []byte) io.Reader {
	return io.MultiReader(bytes.NewReader(data), iotest.ErrReader(errSource))
}

// readSamples reads a decoder until an error, returning the samples before it.
func readSamples(decoder *Decoder) ([][2]float64, error) {
	var out [][2]float64

	buffer := make([][2]float64, 1000)

	for {
		read, err := decoder.Read(buffer)
		out = append(out, buffer[:read]...)

		if err != nil {
			return out, err
		}
	}
}
