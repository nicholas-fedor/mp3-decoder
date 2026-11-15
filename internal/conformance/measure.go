// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

import (
	"errors"
	"fmt"
	"math"
)

// Difference is how far one decoded channel lies from one reference channel.
type Difference struct {
	// RMS is the root mean square of the difference.
	RMS float64
	// Max is the largest difference.
	Max float64
}

// Errors from measuring.
var (
	// ErrChannels is returned for a waveform whose channels are not the reference's.
	ErrChannels = errors.New("conformance: waveform channels differ from the reference's")
	// ErrShort is returned for decoded output that ends before the reference does.
	ErrShort = errors.New("conformance: decoded output ends before the reference")
)

// Full accuracy, as ISO/IEC 11172-4 clause 2.6.3 bounds the difference for samples between -1
// and 1.
var (
	// fullRMS bounds the root mean square difference, 2^-15/√12.
	fullRMS = math.Ldexp(1, -15) / math.Sqrt(12)
	// fullMax bounds the largest difference, 2^-14.
	fullMax = math.Ldexp(1, -14)
)

// loudest is the largest sample a 24-bit reference holds, 1 - 2^-23.
var loudest = float64(fullScale-1) / fullScale

// Measure measures decoded output against the reference, channel by channel.
//
// Each decoded sample is first clipped to the range a 24-bit reference holds, from -1 to just
// under 1, since a stream may decode to louder samples than a reference can hold.
//
// Parameters:
//   - decoded: the stream's decoded samples, as left and right pairs.
//   - wave: the reference's waveform.
//
// Returns:
//   - []Difference: the difference for each channel of the reference.
//   - error: [ErrChannels] or [ErrShort].
func (r Reference) Measure(decoded [][2]float64, wave Waveform) ([]Difference, error) {
	if len(wave.Channels) != len(r.Channels) || len(r.Channels) == 0 {
		return nil, fmt.Errorf("%w: %d, not %d", ErrChannels, len(wave.Channels), len(r.Channels))
	}

	length := len(wave.Channels[0])
	if r.Start+length > len(decoded) {
		return nil, fmt.Errorf("%w: %d samples, not %d", ErrShort, len(decoded), r.Start+length)
	}

	differences := make([]Difference, 0, len(r.Channels))

	for at, want := range wave.Channels {
		var squares, largest float64

		for sample, value := range want {
			difference := min(max(decoded[r.Start+sample][r.Channels[at]], -1), loudest) - value

			squares += float64(difference * difference)

			largest = max(largest, math.Abs(difference))
		}

		differences = append(differences, Difference{RMS: math.Sqrt(squares / float64(max(length, 1))), Max: largest})
	}

	return differences, nil
}

// Full reports whether the difference is within full accuracy, which needs an RMS below 2^-15/√12
// and no difference above 2^-14.
//
// Returns:
//   - bool: true within full accuracy.
func (d Difference) Full() bool {
	return d.RMS < fullRMS && d.Max <= fullMax
}
