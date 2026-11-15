// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3

import (
	"bytes"
	"context"
	"io"
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

// TestCountCanceled checks that a canceled walk over a stream's frames stops with the context's
// error.
func TestCountCanceled(t *testing.T) {
	t.Parallel()

	walker := scanner{source: source{reader: bytes.NewReader(frames(stereoHead, 3))}}

	_, _, err := walker.open(t.Context())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = walker.count(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
