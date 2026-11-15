// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"errors"
	"fmt"
	"io"
)

// source buffers the bytes of a stream that the decoder has read but not used.
type source struct {
	// reader is the stream.
	reader io.Reader
	// buffer holds the bytes read, of which those from at on are unused.
	buffer []byte
	// at is the first unused byte in buffer.
	at int
	// passed counts the bytes of the stream moved past.
	passed int64
	// ended reports whether the reader has returned io.EOF.
	ended bool
}

// readSize is the most bytes one read asks for.
const readSize = 8192

// bytes returns the unused bytes.
//
// Returns:
//   - []byte: the bytes, valid until the next call that reads or moves past bytes.
func (s *source) bytes() []byte {
	return s.buffer[s.at:]
}

// consume moves past a count of buffered bytes.
//
// Parameters:
//   - count: the bytes to move past, at most those buffered.
func (s *source) consume(count int) {
	s.at += count
	s.passed += int64(count)
}

// discard moves past a count of bytes, reading past the buffered ones when it must.
//
// Parameters:
//   - count: the bytes to move past.
//
// Returns:
//   - error: a read error other than [io.EOF].
func (s *source) discard(count int) error {
	buffered := min(count, len(s.buffer)-s.at)
	s.consume(buffered)

	if buffered == count || s.ended {
		return nil
	}

	copied, err := io.CopyN(io.Discard, s.reader, int64(count-buffered))

	s.passed += copied

	switch {
	case errors.Is(err, io.EOF):
		s.ended = true
	case err != nil:
		return fmt.Errorf("read stream: %w", err)
	default:
	}

	return nil
}

// ensure reads until a count of unused bytes is buffered or the stream ends.
//
// Parameters:
//   - count: the bytes wanted.
//
// Returns:
//   - error: a read error other than [io.EOF].
func (s *source) ensure(count int) error {
	for len(s.buffer)-s.at < count && !s.ended {
		unused := copy(s.buffer, s.buffer[s.at:])

		s.buffer, s.at = s.buffer[:unused], 0

		if free := cap(s.buffer) - unused; free < readSize {
			s.buffer = append(s.buffer[:cap(s.buffer)], make([]byte, readSize-free)...)[:unused]
		}

		read, err := s.reader.Read(s.buffer[unused : unused+readSize])

		s.buffer = s.buffer[:unused+read]

		switch {
		case errors.Is(err, io.EOF):
			s.ended = true
		case err != nil:
			return fmt.Errorf("read stream: %w", err)
		default:
		}
	}

	return nil
}
