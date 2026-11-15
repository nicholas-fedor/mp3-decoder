// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"bytes"
	"context"
	"io"
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/mp3test"
)

// TestScanPastUnconfirmable checks that a header in junk whose frame would end past the end of the
// stream stops neither opening nor a resync, when real frames follow it.
//
// The header claims a frame of 1045 bytes at 320 kbit/s, and the two frames after it hold 834. A
// byte of junk before it keeps it from the place where the next frame is expected, where it would
// be the last frame, cut short.
func TestScanPastUnconfirmable(t *testing.T) {
	t.Parallel()

	fake := mp3test.Header{Version: mp3test.MPEG1, Layer: mp3test.Layer3, Bitrate: 14}.Bytes()

	tests := []struct {
		name   string
		stream []byte
		want   int
	}{
		{"opening", slices.Concat(fake, frames(stereoHead, 2)), 2},
		{"resync", slices.Concat(frames(stereoHead, 3), []byte{0}, fake, frames(stereoHead, 2)), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder, err := OpenStream(t.Context(), bytes.NewReader(tt.stream))
			require.NoError(t, err)

			got, err := readSamples(decoder)
			require.ErrorIs(t, err, io.EOF)
			assert.Len(t, got, tt.want*frame.MaxSamples)
		})
	}
}

// TestScannerSeek checks that seeks forward, back and past the end stand the scanner at the frame
// wanted, and that a canceled walk stops with the context's error.
func TestScannerSeek(t *testing.T) {
	t.Parallel()

	data := slices.Concat(frames(stereoHead, 2), []byte("junk"), frames(stereoHead, 3))

	walker := scanner{source: source{reader: bytes.NewReader(data), file: bytes.NewReader(data), end: int64(len(data))}}
	require.NoError(t, walker.open(t.Context()))

	// The frames after the junk start 4 bytes later.
	starts := []int64{0, 417, 838, 1255, 1672}

	for _, target := range []int64{3, 1, 4, 0, 2, 2} {
		require.NoError(t, walker.seek(t.Context(), target))
		assert.Equal(t, target, walker.frames)

		_, _, _, err := walker.frame()
		require.NoError(t, err)
		assert.Equal(t, starts[target]+417, walker.source.passed, "frame %d", target)
	}

	require.NoError(t, walker.seek(t.Context(), math.MaxInt64))
	assert.Equal(t, int64(5), walker.frames)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, walker.seek(ctx, 3), "a seek to an indexed frame walks no frames")

	fresh := scanner{source: source{reader: bytes.NewReader(data), file: bytes.NewReader(data), end: int64(len(data))}}
	require.NoError(t, fresh.open(t.Context()))
	require.ErrorIs(t, fresh.seek(ctx, 3), context.Canceled)
}
