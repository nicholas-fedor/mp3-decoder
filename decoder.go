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
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
)

// Decoder decodes an MPEG audio stream into PCM samples.
type Decoder struct {
	// pending holds the samples of pcm that Read has not handed out.
	pending [][2]float64
	// source buffers the stream.
	source source
	// layer3 decodes the frames.
	layer3 layer3.Decoder
	// pcm holds the samples of the frame decoded last.
	pcm [frame.MaxSamples][2]float64
	// first is the header of the stream's first frame.
	first frame.Header
}

// Errors from opening a stream.
var (
	// ErrNoFrames is returned for a stream in which no frame is found.
	ErrNoFrames = errors.New("mp3: no frames found")
	// ErrLayer is returned for a stream of Layer I or Layer II frames.
	ErrLayer = errors.New("mp3: only Layer III streams are decoded")
)

// OpenStream opens a stream for decoding from its start.
//
// It moves past an ID3v2 tag at the start, then searches for the first frame that the header
// after it confirms. A Xing or Info tag frame that opens the stream holds no audio and is skipped.
// The decoder does not close src.
//
// Parameters:
//   - ctx: cancels the search for the first frame.
//   - src: the stream.
//
// Returns:
//   - *Decoder: the decoder, at the first frame.
//   - error: [ErrNoFrames], [ErrLayer], a read error, or the context's error.
func OpenStream(ctx context.Context, src io.Reader) (*Decoder, error) {
	decoder := new(Decoder)

	decoder.source.reader = src

	err := decoder.source.ensure(container.ID3v2HeaderSize)
	if err != nil {
		return nil, err
	}

	err = decoder.source.discard(container.ID3v2Size(decoder.source.bytes()))
	if err != nil {
		return nil, err
	}

	err = decoder.findFirst(ctx)
	if err != nil {
		return nil, err
	}

	if decoder.first.Layer != frame.Layer3 {
		return nil, ErrLayer
	}

	// The header that confirmed the first frame follows it, so the whole frame is buffered.
	if _, ok := container.ParseInfo(decoder.first, decoder.source.bytes()[:decoder.first.Size()]); ok {
		decoder.source.consume(decoder.first.Size())
	}

	return decoder, nil
}

// Read decodes the next samples into dst.
//
// The stream ends at the first bytes that do not hold a whole frame of the stream that its first
// frame started. A mono stream gives each sample to both channels. The samples are not clipped.
//
// Parameters:
//   - dst: where the samples go, as left and right pairs.
//
// Returns:
//   - int: the number of samples decoded.
//   - error: [io.EOF] after the last sample, unwrapped, or a read or decoding error.
func (d *Decoder) Read(dst [][2]float64) (int, error) {
	decoded := 0

	for decoded < len(dst) {
		if len(d.pending) == 0 {
			err := d.decodeFrame()

			switch {
			case errors.Is(err, io.EOF) && decoded > 0:
				return decoded, nil
			case err != nil:
				return decoded, err
			default:
			}
		}

		copied := copy(dst[decoded:], d.pending)

		d.pending = d.pending[copied:]

		decoded += copied
	}

	return decoded, nil
}

// decodeFrame decodes the next frame into pcm and hands its samples to pending.
//
// A frame that fails to decode is moved past, so the next call decodes the frame after it.
//
// Returns:
//   - error: [io.EOF] at the end of the stream, or a read or decoding error.
func (d *Decoder) decodeFrame() error {
	err := d.source.ensure(frame.HeaderSize)
	if err != nil {
		return err
	}

	head, ok := frame.Parse(d.source.bytes())
	if !ok || !d.first.SameStream(head) {
		return io.EOF
	}

	if head.Free {
		head.Bitrate = d.first.Bitrate
	}

	size := head.Size()

	err = d.source.ensure(size)
	if err != nil {
		return err
	}

	data := d.source.bytes()
	if len(data) < size {
		return io.EOF
	}

	err = d.layer3.Decode(d.pcm[:], head, data[:size])
	d.source.consume(size)

	if err != nil {
		return fmt.Errorf("decode frame: %w", err)
	}

	d.pending = d.pcm[:head.Samples()]

	return nil
}

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

		at, head, ok := frame.Find(d.source.bytes(), nil)
		if ok {
			d.first = head
			d.source.consume(at)

			return nil
		}

		if d.source.ended {
			return ErrNoFrames
		}

		d.source.consume(at)

		err = d.source.ensure(len(d.source.bytes()) + readSize)
		if err != nil {
			return err
		}
	}
}
