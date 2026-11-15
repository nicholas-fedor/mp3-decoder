// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package synth

//go:generate go test -run ^TestSynthTables$ -update
//go:generate go test -run ^TestDCTKernel$ -update

// Filterbank is the polyphase synthesis filterbank of one channel.
//
// It keeps the last 16 vectors that matrixing gave, which the window reaches back over.
type Filterbank struct {
	// history holds the last 16 vectors of 64 values twice over, the newest starting at newest and
	// the older ones after it, so the window reads them without wrapping around.
	history [2 * historySize]float64
	// newest is where the newest vector starts in history.
	newest int
}

// Filterbanks holds the polyphase synthesis filterbank of each channel of a stream.
type Filterbanks struct {
	// Channel holds each channel's filterbank.
	Channel [2]Filterbank
	// channels is the number of channels of the frame before, or 0 before the first.
	channels int
}

// Filterbank sizes.
const (
	// Subbands is the number of polyphase subbands, and the number of PCM samples each set of
	// subband samples gives.
	Subbands = 32
	// vectorSize is the number of values matrixing gives.
	vectorSize = 2 * Subbands
	// historySize is the number of values the filterbank keeps, 16 vectors.
	historySize = 16 * vectorSize
	// windowTaps is the number of taps in the window.
	windowTaps = 512
	// windowGroups is the number of groups of 64 taps in the window.
	windowGroups = windowTaps / vectorSize
	// dctFactors is the number of factors the 32-point DCT takes, 16 + 8 + 4 + 2 + 1.
	dctFactors = Subbands - 1
)

// Follow readies the filterbanks for a frame, and reports whether it started them afresh.
//
// A frame with another number of channels than the frame before it starts both filterbanks afresh,
// as a new stream would.
//
// Parameters:
//   - channels: the frame's number of channels.
//
// Returns:
//   - bool: whether both filterbanks started afresh.
func (f *Filterbanks) Follow(channels int) bool {
	restart := f.channels != 0 && f.channels != channels
	if restart {
		for ch := range f.Channel {
			f.Channel[ch].Reset()
		}
	}

	f.channels = channels

	return restart
}

// Reset forgets the history, as at the start of a stream.
func (f *Filterbank) Reset() {
	f.history = [2 * historySize]float64{}
	f.newest = 0
}

// Synthesize turns one sample of each of the 32 subbands into the next 32 PCM samples.
//
// Matrixing turns the subband samples S(k) into 64 values V(i), the sum over k of
// S(k) cos(π/64 (16+i)(2k+1)). A 32-point DCT of the samples gives them, and they become the
// newest of the 16 vectors kept. Output sample j is then the sum over the 8 groups g of D(64g+j)
// times value j of vector 2g, and D(64g+32+j) times value 32+j of vector 2g+1, counting vectors
// from the newest. D is the 512-tap synthesis window. The samples are not clipped.
//
// Parameters:
//   - dst: the PCM samples.
//   - samples: one sample of each subband.
func (f *Filterbank) Synthesize(dst, samples *[Subbands]float64) {
	f.newest = (f.newest - vectorSize) & (historySize - 1)

	vector := (*[vectorSize]float64)(f.history[f.newest:])
	matrixing(vector, samples)
	copy(f.history[f.newest+historySize:], vector[:])

	*dst = [Subbands]float64{}

	// The groups run outermost, so the 32 sums grow side by side rather than each waiting on its last
	// addition.
	for group := range windowGroups {
		at := f.newest + 2*vectorSize*group
		even := (*[Subbands]float64)(f.history[at:])
		odd := (*[Subbands]float64)(f.history[at+vectorSize+Subbands:])
		taps := (*[vectorSize]float64)(window[vectorSize*group:])

		for sample := range dst {
			dst[sample] += float64(even[sample] * taps[sample])
			dst[sample] += float64(odd[sample] * taps[Subbands+sample])
		}
	}
}

// matrixing turns one sample of each subband into the 64 values V(i) of a vector.
//
// V(i) is the sum over k of S(k) cos(π/64 (16+i)(2k+1)). With X(m), the sum over k of
// S(k) cos(π/64 m(2k+1)), the 32-point DCT of the samples, the angles fold onto those of X, so
// V(i) is X(16+i) for i below 16, 0 at 16, -X(48-i) up to 47, and -X(i-48) from 48 on.
//
// Parameters:
//   - vector: the 64 values.
//   - samples: one sample of each subband.
func matrixing(vector *[vectorSize]float64, samples *[Subbands]float64) {
	spectrum := *samples

	dct32(&spectrum)

	const (
		// fold is the index of V whose angle is π/2, which is 0.
		fold = Subbands / 2
		// mirror is the index of V whose angle is π, from which V counts X down again.
		mirror = 3 * fold
	)

	copy(vector[:fold], spectrum[fold:])

	vector[fold] = 0

	for i := fold + 1; i < mirror; i++ {
		vector[i] = -spectrum[mirror-i]
	}

	for i := mirror; i < vectorSize; i++ {
		vector[i] = -spectrum[i-mirror]
	}
}
