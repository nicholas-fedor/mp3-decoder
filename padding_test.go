// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// TestPaddingOf checks the encoder padding of tags with and without a frame count, with padding
// shorter than the decoder delay, and without the delay and padding.
func TestPaddingOf(t *testing.T) {
	t.Parallel()

	mpeg1 := frame.Header{SampleRate: 44100, Version: frame.MPEG1, Layer: frame.Layer3}
	tagged := func(frames int64, delay, padding int) container.Info {
		return container.Info{Frames: frames, Delay: delay, Padding: padding, Gapless: true}
	}

	tests := []struct {
		name string
		info container.Info
		want encoderPadding
	}{
		{"LAME's defaults", tagged(21, 576, 1566), encoderPadding{skip: 1105, discard: 21*1152 - 1566 + 529, resume: 21 * 1152}},
		{"no frame count", tagged(0, 576, 1566), encoderPadding{skip: 1105, discard: math.MaxInt64, resume: math.MaxInt64}},
		{"padding under the decoder delay", tagged(21, 576, 100), encoderPadding{skip: 1105, discard: 21 * 1152, resume: 21 * 1152}},
		{"no delay and padding", container.Info{Frames: 21}, encoderPadding{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, paddingOf(tt.info, mpeg1))
		})
	}
}

// TestPaddingKeep checks the samples kept of frames at the start, in the middle, at the end, past
// the frames the tag counts, and of frames that the skipped samples cover whole.
func TestPaddingKeep(t *testing.T) {
	t.Parallel()

	lame := encoderPadding{skip: 1105, discard: 23155, resume: 24192}

	tests := []struct {
		name     string
		padding  encoderPadding
		at       int64
		count    int
		from, to int
	}{
		{"no padding", encoderPadding{}, 0, 1152, 0, 1152},
		{"the first frame", lame, 0, 1152, 1105, 1152},
		{"a middle frame", lame, 1152, 1152, 0, 1152},
		{"the last frame the tag counts", lame, 23040, 1152, 0, 115},
		{"a frame past the count", lame, 24192, 1152, 0, 1152},
		{"a frame the skipped samples cover", encoderPadding{skip: 1105, discard: 5000, resume: 5760}, 0, 576, 576, 576},
		{"the frame after it", encoderPadding{skip: 1105, discard: 5000, resume: 5760}, 576, 576, 529, 576},
		{"a skip past the discard", encoderPadding{skip: 2000, discard: 1500, resume: 2304}, 1152, 1152, 848, 848},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			from, to := tt.padding.keep(tt.at, tt.count)

			assert.Equal(t, tt.from, from, "from")
			assert.Equal(t, tt.to, to, "to")
		})
	}
}
