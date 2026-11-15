// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// scanner finds the frames of a stream in its bytes.
//
// It steps over the tags a stream may carry, and passes over damage and junk to the next frame
// that the header after it confirms. Its first frame fixes what the stream's frames must keep. It
// numbers the audio frames it finds from 0, and indexes where it found them.
type scanner struct {
	// index holds where the search for the frames started.
	index frameIndex
	// source buffers the stream.
	source source
	// tag is the stream's Xing, Info or VBRI tag, or the zero value when its first frame holds none.
	tag container.Info
	// first is the header of the stream's first frame.
	first frame.Header
	// frames is the number of the next frame.
	frames int64
}

// findFirst searches the stream for its first confirmed frame and stops at it.
//
// Parameters:
//   - ctx: cancels the search.
//
// Returns:
//   - error: [ErrNoFrames], a read error, or the context's error.
func (s *scanner) findFirst(ctx context.Context) error {
	for {
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("find first frame: %w", err)
		}

		head, found, err := s.scan()

		switch {
		case errors.Is(err, io.EOF):
			return ErrNoFrames
		case err != nil:
			return err
		case found:
			s.first = head

			return nil
		default:
		}
	}
}

// frame moves past the next frame of the stream and returns it.
//
// A last frame that the stream ends inside is returned with the bytes it holds, when it holds its
// side information.
//
// Returns:
//   - frame.Header: the frame's header, with the bitrate of a free-format stream.
//   - []byte: the frame's bytes from its header on, valid until the next call.
//   - bool: whether bytes before the frame were passed over as damage or junk.
//   - error: [io.EOF] at the end of the stream, or a read error.
func (s *scanner) frame() (frame.Header, []byte, bool, error) {
	s.index.add(s.frames, s.source.passed)

	head, resynced, err := s.nextFrame()
	if err != nil {
		return frame.Header{}, nil, false, err
	}

	if head.Free {
		head.Bitrate = s.first.Bitrate
	}

	err = s.source.ensure(head.Size())
	if err != nil {
		return frame.Header{}, nil, false, err
	}

	data := s.source.bytes()

	data = data[:min(len(data), head.Size())]

	if len(data) < head.DataAt() {
		return frame.Header{}, nil, false, io.EOF
	}

	s.source.consume(len(data))

	s.frames++

	return head, data, resynced, nil
}

// nextFrame moves to the next frame of the stream and returns its header.
//
// It steps over the tags a stream may carry between its frames. Any other bytes that are not a
// frame of the stream are damage or junk, and a resync passes over them.
//
// Returns:
//   - frame.Header: the frame's header.
//   - bool: whether a resync passed over bytes before the frame.
//   - error: [io.EOF] at the end of the stream, or a read error.
func (s *scanner) nextFrame() (frame.Header, bool, error) {
	for {
		err := s.source.ensure(frame.HeaderSize)
		if err != nil {
			return frame.Header{}, false, err
		}

		head, ok := frame.Parse(s.source.bytes())
		if ok && s.first.SameStream(head) {
			return head, false, nil
		}

		stepped, err := s.skipTag()

		switch {
		case err != nil:
			return frame.Header{}, false, err
		case !stepped:
			head, err = s.resync()

			return head, err == nil, err
		default:
		}
	}
}

// open moves past the tags at the start of the stream to its first confirmed frame, and past that
// frame too when it holds a Xing, Info or VBRI tag, which it keeps.
//
// Parameters:
//   - ctx: cancels the search for the first frame.
//
// Returns:
//   - error: [ErrNoFrames], a read error, or the context's error.
func (s *scanner) open(ctx context.Context) error {
	for stepped := true; stepped; {
		var err error

		stepped, err = s.skipTag()
		if err != nil {
			return err
		}
	}

	err := s.findFirst(ctx)
	if err != nil {
		return err
	}

	// The header that confirmed the first frame follows it, so the whole frame is buffered.
	info, ok := container.ParseInfo(s.first, s.source.bytes()[:s.first.Size()])
	if ok {
		s.tag = info
		s.source.consume(s.first.Size())
	}

	return nil
}

// resync passes over bytes that are not a frame of the stream, to the next frame that the header
// after it confirms.
//
// The stream ends where a confirmed frame of another stream starts.
//
// Returns:
//   - frame.Header: the frame's header.
//   - error: [io.EOF] at the end of the stream, or a read error.
func (s *scanner) resync() (frame.Header, error) {
	for {
		head, found, err := s.scan()

		switch {
		case err != nil:
			return frame.Header{}, err
		case !found:
			continue
		case !s.first.SameStream(head):
			return frame.Header{}, io.EOF
		default:
			return head, nil
		}
	}
}

// scan searches the buffered bytes for a frame that the header after it confirms and stops at it,
// or else moves past the bytes searched and reads more.
//
// The search stops early at a header whose frame the bytes end before confirming. When the stream
// has ended, no more bytes can confirm it, so the search moves past it.
//
// Returns:
//   - frame.Header: the frame's header, with the bitrate a free-format frame's spacing gives.
//   - bool: whether a frame was found.
//   - error: [io.EOF] when the stream ends before a frame is found, or a read error.
func (s *scanner) scan() (frame.Header, bool, error) {
	at, head, ok := frame.Find(s.source.bytes(), nil)
	s.source.consume(at)

	switch {
	case ok:
		return head, true, nil
	case s.source.ended && len(s.source.bytes()) >= frame.HeaderSize:
		s.source.consume(1)

		return frame.Header{}, false, nil
	case s.source.ended:
		return frame.Header{}, false, io.EOF
	default:
		return frame.Header{}, false, s.source.ensure(len(s.source.bytes()) + readSize)
	}
}

// seek moves to a frame of a file, or to the end of the stream when it holds fewer frames.
//
// It starts from the last indexed frame at or before the one wanted, unless the stream already
// stands between them. It then moves past the frames up to the one wanted.
//
// Parameters:
//   - ctx: cancels the walk.
//   - target: the frame's number.
//
// Returns:
//   - error: a read or seek error, or the context's error.
func (s *scanner) seek(ctx context.Context, target int64) error {
	from, offset, ok := s.index.before(target)
	if ok && (target < s.frames || from > s.frames) {
		err := s.source.seek(offset)
		if err != nil {
			return err
		}

		s.frames = from
	}

	for s.frames < target {
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("seek frame: %w", err)
		}

		_, _, _, err = s.frame()

		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		default:
		}
	}

	return nil
}

// skipTag moves past a tag that starts at the stream's position.
//
// Returns:
//   - bool: whether a tag was there.
//   - error: a read error.
func (s *scanner) skipTag() (bool, error) {
	err := s.source.ensure(container.TagHeadSize)
	if err != nil {
		return false, err
	}

	size := container.TagSize(s.source.bytes())
	if size == 0 {
		return false, nil
	}

	return true, s.source.discard(size)
}
