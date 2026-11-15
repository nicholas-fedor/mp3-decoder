// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"math"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// encoderPadding holds the samples an encoder added at the start and the end of a stream, which
// decoding leaves out so the stream plays without gaps.
//
// Samples count from the first sample of the first frame after the tag frame. The zero value
// leaves out no sample.
type encoderPadding struct {
	// skip is the number of samples skipped at the start.
	skip int64
	// discard is the first sample discarded at the end.
	discard int64
	// resume is the first sample after those discarded, from which samples are kept again.
	resume int64
}

// decoderDelay is the number of samples by which a decoder's filterbanks delay its output, which
// encoders add to the delay their tag gives.
const decoderDelay = 529

// paddingOf returns the encoder padding of a stream whose tag frame gives its encoder delay and
// padding.
//
// The start skips the encoder delay and the decoder delay. With the tag's frame count known, the
// end discards the padding, less the decoder delay, before the last of those frames ends. Frames
// past the count keep their samples.
//
// Parameters:
//   - info: the stream's tag.
//   - head: the header of the tag frame.
//
// Returns:
//   - encoderPadding: the padding, or the zero value when the tag gives no delay and padding.
func paddingOf(info container.Info, head frame.Header) encoderPadding {
	if !info.Gapless {
		return encoderPadding{skip: 0, discard: 0, resume: 0}
	}

	padding := encoderPadding{skip: int64(info.Delay) + decoderDelay, discard: math.MaxInt64, resume: math.MaxInt64}

	if info.Frames > 0 {
		padding.resume = info.Frames * int64(head.Samples())
		padding.discard = min(padding.resume-int64(info.Padding)+decoderDelay, padding.resume)
	}

	return padding
}

// keep returns the part of a frame's samples that is not padding.
//
// Parameters:
//   - at: the frame's first sample.
//   - count: the number of samples the frame holds.
//
// Returns:
//   - from: the first sample kept, from the frame's start.
//   - to: the sample after the last kept.
//
//nolint:nonamedreturns // Same-type returns need names.
func (p encoderPadding) keep(at int64, count int) (from, to int) {
	if at >= p.resume {
		return 0, count
	}

	from = int(min(max(p.skip-at, 0), int64(count)))
	to = int(min(max(p.discard-at, int64(from)), int64(count)))

	return from, to
}

// kept returns how many samples of a stream are not padding.
//
// Parameters:
//   - total: the number of samples the stream's frames hold.
//
// Returns:
//   - int64: the number of samples not left out.
func (p encoderPadding) kept(total int64) int64 {
	return max(min(p.discard, total)-p.skip, 0) + max(total-p.resume, 0)
}
