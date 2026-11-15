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
	// dctFactors is the number of factors the 32-point DCT takes, 16 + 8 + 4 + 2 + 1.
	dctFactors = Subbands - 1
	// dctScratch is the number of values the 32-point DCT keeps while it runs, 32 + 16 + 8 + 4 + 2.
	dctScratch = 2*Subbands - 2
)

// Reset forgets the history, as at the start of a stream.
func (f *Filterbank) Reset() {
	f.history = [historySize]float64{}
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

	matrixing((*[vectorSize]float64)(f.history[f.newest:f.newest+vectorSize]), samples)

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
	var scratch [dctScratch]float64

	spectrum := *samples

	dct(spectrum[:], scratch[:])

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

// dct turns values into their DCT in place, by Byeong Gi Lee's algorithm.
//
// Of n values x(k), it gives X(m), the sum over k of x(k) cos(π m(2k+1)/(2n)). It splits them into
// the sums g(k) = x(k) + x(n-1-k) and the scaled differences h(k) = (x(k) - x(n-1-k)) /
// (2 cos(π(2k+1)/(2n))), whose DCTs of n/2 points G and H give X(2m) = G(m) and
// X(2m+1) = H(m) + H(m+1), with H(n/2) taken as 0.
//
// Parameters:
//   - values: the values, a power of two of them up to 32.
//   - scratch: room for the values of every split, 2n - 2 of them.
func dct(values, scratch []float64) {
	points := len(values)
	half := points >> 1
	split := scratch[:points]
	factors := lee[Subbands-points : Subbands-half]

	for k := range half {
		a, b := values[k], values[points-1-k]

		split[k] = a + b
		split[half+k] = float64((a - b) * factors[k])
	}

	if half > 1 {
		dct(split[:half], scratch[points:])
		dct(split[half:], scratch[points:])
	}

	for m := range half - 1 {
		values[2*m] = split[m]
		values[2*m+1] = split[half+m] + split[half+m+1]
	}

	values[points-2] = split[half-1]
	values[points-1] = split[points-1]
}
