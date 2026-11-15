// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer12"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
)

// Decoder decodes an MPEG audio stream into PCM samples.
type Decoder struct {
	// pending holds the samples of pcm that Read has not handed out.
	pending [][2]float64
	// scanner finds the stream's frames.
	scanner scanner
	// layer12 decodes the frames of a Layer I or Layer II stream.
	layer12 layer12.Decoder
	// layer3 decodes the frames of a Layer III stream.
	layer3 layer3.Decoder
	// pcm holds the samples of the frame decoded last.
	pcm [frame.MaxSamples][2]float64
	// stats counts what decoding has met.
	stats Stats
	// padding holds the samples the encoder added, which are left out.
	padding encoderPadding
	// length is the number of samples the stream decodes to, when known.
	length int64
	// cut holds a last frame that the stream ends inside, with zeros for the bytes it lacks.
	cut [frame.MaxSize]byte
	// known reports whether length is known.
	known bool
}

// Info describes a stream, as its first frame gives it.
type Info struct {
	// SampleRate is the sampling frequency in Hz.
	SampleRate int
	// Channels is the number of channels, 1 or 2. Read gives each sample of a mono stream to both
	// channels.
	Channels int
}

// Stats counts what decoding a stream has met.
//
// Damage never ends a stream. Each count says how often decoding carried on past one kind of it.
type Stats struct {
	// Frames is the number of frames decoded.
	Frames int64
	// Resyncs is the number of times bytes that are not a frame of the stream were passed over.
	Resyncs int64
	// CRCMismatches is the number of frames whose CRC does not match. They are decoded all the same.
	CRCMismatches int64
	// Underflows is the number of frames whose main data began before the bytes the decoder held,
	// which give silence.
	Underflows int64
	// Undecodable is the number of frames that hold a value no encoder writes, which give silence.
	Undecodable int64
}

// tailSize is the most bytes at the end of a file searched for the tags after its audio.
const tailSize = 64 << 10

// ErrNoFrames is returned for a stream in which no frame is found.
var ErrNoFrames = errors.New("mp3: no frames found")

// Errors from seeking.
var (
	// ErrNotSeekable is returned for a seek on a stream that [OpenStream] opened.
	ErrNotSeekable = errors.New("mp3: only a file from Open can seek")
	// ErrOutOfRange is returned for a seek to a sample outside the stream.
	ErrOutOfRange = errors.New("mp3: sample outside the stream")
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
//   - error: [ErrNoFrames], a read or seek error, or the context's error.
func Open(ctx context.Context, src io.ReadSeeker) (*Decoder, error) {
	end, err := audioEnd(src)
	if err != nil {
		return nil, err
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek stream: %w", err)
	}

	decoder, err := OpenStream(ctx, io.LimitReader(src, end))
	if err != nil {
		return nil, err
	}

	stream := &decoder.scanner

	stream.source.file, stream.source.end = src, end

	frames := stream.tag.Frames

	if frames == 0 || stream.tag.Bytes != end-stream.source.passed+int64(stream.first.Size()) {
		err = stream.seek(ctx, math.MaxInt64)
		if err != nil {
			return nil, err
		}

		frames = stream.frames

		err = stream.seek(ctx, 0)
		if err != nil {
			return nil, err
		}
	}

	decoder.length, decoder.known = decoder.padding.kept(frames*int64(stream.first.Samples())), true

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
//   - error: [ErrNoFrames], a read error, or the context's error.
func OpenStream(ctx context.Context, src io.Reader) (*Decoder, error) {
	decoder := new(Decoder)

	decoder.scanner.source.reader = src

	err := decoder.scanner.open(ctx)
	if err != nil {
		return nil, err
	}

	decoder.padding = paddingOf(decoder.scanner.tag, decoder.scanner.first)

	return decoder, nil
}

// Info returns what the stream's first frame says of it.
//
// Returns:
//   - Info: the sampling frequency and the number of channels.
func (d *Decoder) Info() Info {
	return Info{SampleRate: d.scanner.first.SampleRate, Channels: d.scanner.first.Channels()}
}

// Len returns the number of samples the stream decodes to.
//
// Only [Open] finds it.
//
// Returns:
//   - int64: the number of samples, from the first kept.
//   - bool: whether the number is known.
func (d *Decoder) Len() (int64, bool) {
	return d.length, d.known
}

// Position returns the number of the sample that Read gives next, from 0 for the first kept.
//
// Returns:
//   - int64: the sample's number.
func (d *Decoder) Position() int64 {
	return d.padding.kept(d.scanner.frames*int64(d.scanner.first.Samples())) - int64(len(d.pending))
}

// Read decodes the next samples into dst.
//
// ID3v2, ID3v1 and APEv2 tags between frames are stepped over. Other bytes that are not a frame of
// the stream, such as damage or junk, are passed over to the next frame that the header after it
// confirms. A Layer III frame whose main data began before them gives silence. So does a frame
// that fails to decode, and every frame thus keeps its place. The stream ends where a frame of
// another stream starts, or where src ends. A last frame that src ends inside is decoded from the
// bytes it holds, with zeros for the rest, when it holds everything before its audio data. A mono
// stream gives each sample to both channels. The samples are not clipped.
//
// When a Layer III stream's tag gives the encoder delay and padding, the samples the encoder added
// are left out. At the start these are the encoder delay and the 529 samples by which decoding
// delays the audio. At the end they are the padding, less those 529 samples, before the last frame the tag
// counts ends. The samples that remain are those the encoder took in.
//
// Parameters:
//   - dst: where the samples go, as left and right pairs.
//
// Returns:
//   - int: the number of samples decoded.
//   - error: [io.EOF] after the last sample, unwrapped, or a read error.
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

// Seek moves to a sample, so that Read gives it next.
//
// The samples Read gives from there are those a Read from the start gives, bit for bit. To restore
// the bit reservoir and the filterbanks, decoding starts some frames before the sample's frame. It
// starts 2 frames before in Layer I and 1 in Layer II. In Layer III it starts up to 10 frames
// before in MPEG-1 and up to 257 at the lowest bitrate of MPEG-2. An index
// of where the frames lie finds them. Reading the file fills it, and a seek past the frames
// indexed walks the frames in between. Seeking to the number of samples [Decoder.Len] gives moves
// to the end.
//
// Parameters:
//   - ctx: cancels the walk.
//   - sample: the sample's number, from 0 for the first kept.
//
// Returns:
//   - error: [ErrNotSeekable], [ErrOutOfRange], a read or seek error, or the context's error.
func (d *Decoder) Seek(ctx context.Context, sample int64) error {
	if d.scanner.source.file == nil {
		return ErrNotSeekable
	}

	if sample < 0 || sample > d.length {
		return fmt.Errorf("seek to sample %d of %d: %w", sample, d.length, ErrOutOfRange)
	}

	target := d.padding.locate(sample)
	samples := int64(d.scanner.first.Samples())

	err := d.scanner.seek(ctx, max(target/samples-int64(d.scanner.first.Preroll()), 0))
	if err != nil {
		return err
	}

	d.layer3.Reset()
	d.layer12.Reset()

	d.pending = nil

	counted := d.stats

	for d.scanner.frames*samples <= target {
		if d.scanner.frames == target/samples {
			d.stats = counted
		}

		err = d.decodeFrame()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}
	}

	// At the end of the stream, all the samples of the frame decoded last come before the sample. A
	// tag whose frame count is wrong can also put the sample outside that frame.
	d.pending = d.pending[min(max(sample-d.Position(), 0), int64(len(d.pending))):]

	return nil
}

// Stats returns the counts of what decoding has met since the stream was opened.
//
// The frames a seek decodes before its sample are not counted.
//
// Returns:
//   - Stats: the counts.
func (d *Decoder) Stats() Stats {
	return d.stats
}

// decodeFrame decodes the next frame into pcm and hands its samples to pending.
//
// Returns:
//   - error: [io.EOF] at the end of the stream, or a read error.
func (d *Decoder) decodeFrame() error {
	head, data, resynced, err := d.scanner.frame()
	if err != nil {
		return err
	}

	d.stats.Frames++

	if resynced {
		d.stats.Resyncs++
		d.layer3.Resync()
	}

	if len(data) < head.Size() {
		// The stream ends inside the frame. Its audio data may still lie in the bytes it holds.
		cut := d.cut[:head.Size()]
		clear(cut[copy(cut, data):])

		data = cut
	}

	var intact bool

	whole := true

	if head.Layer == frame.Layer3 {
		intact, whole, err = d.layer3.Decode(d.pcm[:], head, data)
	} else {
		intact, err = d.layer12.Decode(d.pcm[:], head, data)
	}

	if !intact {
		d.stats.CRCMismatches++
	}

	switch {
	case err != nil:
		// The frame gives silence, so the frames after it keep their samples' numbers.
		clear(d.pcm[:head.Samples()])

		d.stats.Undecodable++
	case !whole:
		d.stats.Underflows++
	default:
	}

	from, to := d.padding.keep((d.scanner.frames-1)*int64(head.Samples()), head.Samples())

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
