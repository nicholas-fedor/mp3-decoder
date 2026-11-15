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

// findFirst searches the stream for its first confirmed frame and stops at it.
//
// Parameters:
//   - ctx: cancels the search.
//
// Returns:
//   - error: [ErrNoFrames], a read error, or the context's error.
func (d *Decoder) findFirst(ctx context.Context) error {
	for {
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("find first frame: %w", err)
		}

		head, found, err := d.scan()

		switch {
		case errors.Is(err, io.EOF):
			return ErrNoFrames
		case err != nil:
			return err
		case found:
			d.first = head

			return nil
		default:
		}
	}
}

// nextFrame moves to the next frame of the stream and returns its header.
//
// It steps over the tags a stream may carry between its frames. Any other bytes that are not a
// frame of the stream are damage or junk, and a resync passes over them.
//
// Returns:
//   - frame.Header: the frame's header.
//   - error: [io.EOF] at the end of the stream, or a read error.
func (d *Decoder) nextFrame() (frame.Header, error) {
	for {
		err := d.source.ensure(frame.HeaderSize)
		if err != nil {
			return frame.Header{}, err
		}

		head, ok := frame.Parse(d.source.bytes())
		if ok && d.first.SameStream(head) {
			return head, nil
		}

		stepped, err := d.skipTag()

		switch {
		case err != nil:
			return frame.Header{}, err
		case !stepped:
			return d.resync()
		default:
		}
	}
}

// resync passes over bytes that are not a frame of the stream, to the next frame that the header
// after it confirms.
//
// The bytes passed over may hold main data that the next frames reach back to, so it empties the
// bit reservoir. The stream ends where a confirmed frame of another stream starts.
//
// Returns:
//   - frame.Header: the frame's header.
//   - error: [io.EOF] at the end of the stream, or a read error.
func (d *Decoder) resync() (frame.Header, error) {
	for {
		head, found, err := d.scan()

		switch {
		case err != nil:
			return frame.Header{}, err
		case !found:
			continue
		case !d.first.SameStream(head):
			return frame.Header{}, io.EOF
		default:
			d.layer3.Resync()

			return head, nil
		}
	}
}

// scan searches the buffered bytes for a frame that the header after it confirms and stops at it,
// or else moves past the bytes searched and reads more.
//
// Returns:
//   - frame.Header: the frame's header, with the bitrate a free-format frame's spacing gives.
//   - bool: whether a frame was found.
//   - error: [io.EOF] when the stream ends before a frame is found, or a read error.
func (d *Decoder) scan() (frame.Header, bool, error) {
	at, head, ok := frame.Find(d.source.bytes(), nil)
	d.source.consume(at)

	switch {
	case ok:
		return head, true, nil
	case d.source.ended:
		return frame.Header{}, false, io.EOF
	default:
		return frame.Header{}, false, d.source.ensure(len(d.source.bytes()) + readSize)
	}
}

// skipTag moves past a tag that starts at the stream's position.
//
// Returns:
//   - bool: whether a tag was there.
//   - error: a read error.
func (d *Decoder) skipTag() (bool, error) {
	err := d.source.ensure(container.TagHeadSize)
	if err != nil {
		return false, err
	}

	size := container.TagSize(d.source.bytes())
	if size == 0 {
		return false, nil
	}

	return true, d.source.discard(size)
}
