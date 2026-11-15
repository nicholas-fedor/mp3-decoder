// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMeasure measures decoded output against references that start part way in and that take the
// right channel, and checks the RMS and the largest difference of each channel.
//
// Decoded samples past full scale match a reference clipped there.
func TestMeasure(t *testing.T) {
	t.Parallel()

	decoded := [][2]float64{{1.5, -2}, {0.5, -0.5}, {0.25, -0.25}, {0, 0.125}, {-1, 0.75}}

	tests := []struct {
		name string
		ref  Reference
		wave Waveform
		want []Difference
	}{
		{
			name: "stereo from the start",
			ref:  Reference{Name: "", MD5: "", Channels: []int{0, 1}, Start: 1, Alternative: false},
			wave: Waveform{Channels: [][]float64{{0.5, 0.25, 0, -1}, {-0.5, -0.25, 0.125, 0.75}}, Rate: 0},
			want: []Difference{{RMS: 0, Max: 0}, {RMS: 0, Max: 0}},
		},
		{
			name: "right channel part way in",
			ref:  Reference{Name: "", MD5: "", Channels: []int{1}, Start: 3, Alternative: false},
			wave: Waveform{Channels: [][]float64{{0.375, 0}}, Rate: 0},
			want: []Difference{{RMS: math.Sqrt((0.0625 + 0.5625) / 2), Max: 0.75}},
		},
		{
			name: "clipped",
			ref:  Reference{Name: "", MD5: "", Channels: []int{0, 1}, Start: 0, Alternative: false},
			wave: Waveform{Channels: [][]float64{{1 - 1.0/fullScale}, {-1}}, Rate: 0},
			want: []Difference{{RMS: 0, Max: 0}, {RMS: 0, Max: 0}},
		},
		{
			name: "empty",
			ref:  Reference{Name: "", MD5: "", Channels: []int{0}, Start: 5, Alternative: false},
			wave: Waveform{Channels: [][]float64{{}}, Rate: 0},
			want: []Difference{{RMS: 0, Max: 0}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.ref.Measure(decoded, test.wave)
			require.NoError(t, err)
			assert.InDeltaSlice(t, rmsOf(test.want), rmsOf(got), 1e-15)
			assert.InDeltaSlice(t, maxOf(test.want), maxOf(got), 1e-15)
		})
	}
}

// TestMeasureErrors checks that a waveform with other channels than the reference's, and decoded
// output that ends before the reference, are refused.
func TestMeasureErrors(t *testing.T) {
	t.Parallel()

	decoded := make([][2]float64, 4)

	tests := []struct {
		name string
		ref  Reference
		wave Waveform
		want error
	}{
		{
			name: "channels",
			ref:  Reference{Name: "", MD5: "", Channels: []int{0, 1}, Start: 0, Alternative: false},
			wave: Waveform{Channels: [][]float64{{0}}, Rate: 0},
			want: ErrChannels,
		},
		{
			name: "no channels",
			ref:  Reference{Name: "", MD5: "", Channels: nil, Start: 0, Alternative: false},
			wave: Waveform{Channels: nil, Rate: 0},
			want: ErrChannels,
		},
		{
			name: "short",
			ref:  Reference{Name: "", MD5: "", Channels: []int{0}, Start: 2, Alternative: false},
			wave: Waveform{Channels: [][]float64{{0, 0, 0}}, Rate: 0},
			want: ErrShort,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := test.ref.Measure(decoded, test.wave)
			require.ErrorIs(t, err, test.want)
		})
	}
}

// TestFull checks full accuracy at its bounds.
//
// An RMS just below 2^-15/√12 and a largest difference of exactly 2^-14 pass, and anything past
// either fails.
func TestFull(t *testing.T) {
	t.Parallel()

	rms := math.Ldexp(1, -15) / math.Sqrt(12)
	largest := math.Ldexp(1, -14)

	assert.True(t, Difference{RMS: math.Nextafter(rms, 0), Max: largest}.Full())
	assert.False(t, Difference{RMS: rms, Max: largest}.Full())
	assert.False(t, Difference{RMS: 0, Max: math.Nextafter(largest, 1)}.Full())
}

// rmsOf returns the RMS of each difference.
func rmsOf(differences []Difference) []float64 {
	values := make([]float64, 0, len(differences))

	for _, difference := range differences {
		values = append(values, difference.RMS)
	}

	return values
}

// maxOf returns the largest difference of each difference.
func maxOf(differences []Difference) []float64 {
	values := make([]float64, 0, len(differences))

	for _, difference := range differences {
		values = append(values, difference.Max)
	}

	return values
}
