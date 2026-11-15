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
	// scanner finds the stream's frames.
	scanner scanner
	// layer3 decodes the frames.
	layer3 layer3.Decoder
	// pcm holds the samples of the frame decoded last.
	pcm [frame.MaxSamples][2]float64
	// cut holds a last frame that the stream ends inside, with zeros for the bytes it lacks.
	cut [frame.MaxSize]byte
	// padding holds the samples the encoder added, which are left out.
	padding encoderPadding
	// decoded counts the samples of the frames decoded, those left out among them.
	decoded int64
	// length is the number of samples the stream decodes to, when known.
	length int64
	// known reports whether length is known.
	known bool
}

// tailSize is the most bytes at the end of a file searched for the tags after its audio.
const tailSize = 64 << 10

// Errors from opening a stream.
var (
	// ErrNoFrames is returned for a stream in which no frame is found.
	ErrNoFrames = errors.New("mp3: no frames found")
	// ErrLayer is returned for a stream of Layer I or Layer II frames.
	ErrLayer = errors.New("mp3: only Layer III streams are decoded")
)

// Open opens a file for decoding from its start, and finds the number of samples it decodes to.
//
// It finds the audio as [OpenStream] does, and decodes only the bytes before the ID3v1, APEv2 and
// Lyrics3 tags at the end. When the stream's tag frame counts its frames and its bytes, and the
// byte count matches the audio, the frame count gives the length. Otherwise a walk over the frame
// headers counts the frames, which reads the whole file. The decoder does not close src.
//
// Parameters:
//   - ctx: cancels the search for the first frame and the walk.
//   - src: the file.
//
// Returns:
//   - *Decoder: the decoder, at the first frame.
//   - error: [ErrNoFrames], [ErrLayer], a read or seek error, or the context's error.
func Open(ctx context.Context, src io.ReadSeeker) (*Decoder, error) {
	end, err := audioEnd(src)
	if err != nil {
		return nil, err
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek stream: %w", err)
	}

	var walker scanner

	walker.source.reader = io.LimitReader(src, end)

	info, tagged, err := walker.open(ctx)
	if err != nil {
		return nil, err
	}

	total := info.Frames * int64(walker.first.Samples())

	if !tagged || info.Frames == 0 || info.Bytes != end-walker.source.passed+int64(walker.first.Size()) {
		total, err = walker.count(ctx)
		if err != nil {
			return nil, err
		}
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek stream: %w", err)
	}

	decoder, err := OpenStream(ctx, io.LimitReader(src, end))
	if err != nil {
		return nil, err
	}

	decoder.length, decoder.known = decoder.padding.kept(total), true

	return decoder, nil
}

// OpenStream opens a stream for decoding from its start.
//
// It moves past the ID3v2, ID3v1 and APEv2 tags at the start, then searches for the first frame
// that the header after it confirms. A Xing or Info tag frame that opens the stream holds no audio
// and is skipped. When its LAME extension gives the encoder delay and padding, the samples the
// encoder added are left out, as Read describes. The decoder does not close src.
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

	decoder.scanner.source.reader = src

	info, tagged, err := decoder.scanner.open(ctx)
	if err != nil {
		return nil, err
	}

	if decoder.scanner.first.Layer != frame.Layer3 {
		return nil, ErrLayer
	}

	if tagged {
		decoder.padding = paddingOf(info, decoder.scanner.first)
	}

	return decoder, nil
}

// Len returns the number of samples the stream decodes to.
//
// Only [Open] finds it. It counts every frame, so a frame that fails to decode makes the stream
// give fewer samples.
//
// Returns:
//   - int64: the number of samples, from the first kept.
//   - bool: whether the number is known.
func (d *Decoder) Len() (int64, bool) {
	return d.length, d.known
}

// Read decodes the next samples into dst.
//
// ID3v2, ID3v1 and APEv2 tags between frames are stepped over. Other bytes that are not a frame of
// the stream, such as damage or junk, are passed over to the next frame that the header after it
// confirms. A frame whose main data began before them gives silence. The stream ends where a frame
// of another stream starts, or where src ends. A last frame that src ends inside is decoded from
// the bytes it holds, with zeros for the rest, when it holds its side information. A mono stream
// gives each sample to both channels. The samples are not clipped.
//
// When the stream's tag gives the encoder delay and padding, the samples the encoder added are left
// out. At the start these are the encoder delay and the 529 samples by which decoding delays the
// audio. At the end they are the padding, less those 529 samples, before the last frame the tag
// counts ends. The samples that remain are those the encoder took in.
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
	head, data, resynced, err := d.scanner.frame()
	if err != nil {
		return err
	}

	if resynced {
		d.layer3.Resync()
	}

	if len(data) < head.Size() {
		// The stream ends inside the frame. Its main data may still lie in the bytes it holds.
		cut := d.cut[:head.Size()]
		clear(cut[copy(cut, data):])

		data = cut
	}

	err = d.layer3.Decode(d.pcm[:], head, data)

	at := d.decoded

	d.decoded += int64(head.Samples())

	if err != nil {
		return fmt.Errorf("decode frame: %w", err)
	}

	from, to := d.padding.keep(at, head.Samples())

	d.pending = d.pcm[from:to]

	return nil
}

// audioEnd returns where the audio of a file ends, before the tags after it.
//
// Parameters:
//   - src: the file.
//
// Returns:
//   - int64: the size of the file less that of the tags at its end.
//   - error: a read or seek error.
func audioEnd(src io.ReadSeeker) (int64, error) {
	size, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("seek stream: %w", err)
	}

	_, err = src.Seek(size-min(size, tailSize), io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("seek stream: %w", err)
	}

	tail, err := io.ReadAll(src)
	if err != nil {
		return 0, fmt.Errorf("read stream: %w", err)
	}

	return size - int64(container.TrailerSize(tail)), nil
}
