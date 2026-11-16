// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// TestJointBound checks the first shared subband of each mode and mode extension.
func TestJointBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head frame.Header
		want int
	}{
		{"mono", frame.Header{Mode: frame.Mono, Extension: 2}, 32},
		{"stereo", frame.Header{Mode: frame.Stereo, Extension: 1}, 32},
		{"dual channel", frame.Header{Mode: frame.DualChannel, Extension: 3}, 32},
		{"joint stereo, extension 0", frame.Header{Mode: frame.JointStereo, Extension: 0}, 4},
		{"joint stereo, extension 1", frame.Header{Mode: frame.JointStereo, Extension: 1}, 8},
		{"joint stereo, extension 2", frame.Header{Mode: frame.JointStereo, Extension: 2}, 12},
		{"joint stereo, extension 3", frame.Header{Mode: frame.JointStereo, Extension: 3}, 16},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, jointBound(tt.head), tt.name)
	}
}
