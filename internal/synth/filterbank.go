// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package synth

//go:generate go test -run ^TestSynthTables$ -update

// Filterbank is the polyphase synthesis filterbank of one channel.
//
// It keeps the last 16 vectors that matrixing gave, which the window reaches back over.
type Filterbank struct {
	// history holds the last 16 vectors of 64 values, the newest starting at newest and the older
	// ones after it, wrapping around.
	history [historySize]float64
	// newest is where the newest vector starts in history.
	newest int
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
)

// Reset forgets the history, as at the start of a stream.
func (f *Filterbank) Reset() {
	f.history = [historySize]float64{}
	f.newest = 0
}

// Synthesize turns one sample of each of the 32 subbands into the next 32 PCM samples.
//
// Matrixing turns the subband samples S(k) into 64 values V(i), the sum over k of
// S(k) cos(π/64 (16+i)(2k+1)), which become the newest of the 16 vectors kept. Output sample j is
// then the sum over the 8 groups g of D(64g+j) times value j of vector 2g, and D(64g+32+j) times
// value 32+j of vector 2g+1, counting vectors from the newest. D is the 512-tap synthesis window.
// The samples are not clipped.
//
// Parameters:
//   - dst: the PCM samples.
//   - samples: one sample of each subband.
func (f *Filterbank) Synthesize(dst, samples *[Subbands]float64) {
	f.newest = (f.newest - vectorSize) & (historySize - 1)

	vector := f.history[f.newest : f.newest+vectorSize]

	for i := range vector {
		cosines := &matrix[i]

		var sum float64

		for k, value := range samples {
			sum += float64(value * cosines[k])
		}

		vector[i] = sum
	}

	for sample := range dst {
		var sum float64

		for group := range windowGroups {
			even := f.newest + 2*vectorSize*group + sample
			odd := even + vectorSize + Subbands

			sum += float64(f.history[even&(historySize-1)] * window[vectorSize*group+sample])
			sum += float64(f.history[odd&(historySize-1)] * window[vectorSize*group+Subbands+sample])
		}

		dst[sample] = sum
	}
}
