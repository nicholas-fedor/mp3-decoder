// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package hybrid

//go:generate go test -run ^TestHybridTables$ -update

// Window is the window the long blocks of a granule take.
type Window uint8

// Block says which subbands of a granule hold long blocks, and the windows they take.
type Block struct {
	// Long is the number of subbands, from 0 to 32, that hold long blocks from the lowest up. The
	// subbands above them hold short blocks.
	Long int
	// Mixed is the number of subbands, from the lowest up, that hold the long part of a mixed
	// block. Their long blocks take the normal window whatever Window says.
	Mixed int
	// Window is the window of the long blocks above the long part of a mixed block.
	Window Window
}

// Filterbank turns one channel's granules of spectral lines into subband samples.
//
// It keeps the second half of each subband's last block, which overlaps the first half of the
// next granule's block.
type Filterbank struct {
	// overlap holds the second half of each subband's last block.
	overlap [Subbands][Samples]float64
}

// Windows of long blocks.
const (
	// Normal is the window of a long block between long blocks.
	Normal Window = iota
	// Start is the window of a long block before short blocks.
	Start
	// Stop is the window of a long block after short blocks.
	Stop
)

// Filterbank sizes.
const (
	// Subbands is the number of polyphase subbands.
	Subbands = 32
	// Samples is the number of samples each subband takes from a granule.
	Samples = 18
	// granuleLines is the number of spectral lines in a granule.
	granuleLines = Subbands * Samples
	// longPoints is the number of samples in a long block.
	longPoints = 2 * Samples
	// shortPoints is the number of samples in a short block.
	shortPoints = 12
	// shortWindows is the number of short blocks in a subband.
	shortWindows = 3
	// shortStep is how far each short block starts after the one before, the first 6 samples into
	// the long block.
	shortStep = 6
)

// Reset forgets the overlap, as at the start of a stream.
func (f *Filterbank) Reset() {
	f.overlap = [Subbands][Samples]float64{}
}

// Transform turns a granule's spectral lines into the next 18 samples of each of the 32 subbands.
//
// Alias reduction runs in place in lines, across each boundary between two subbands of long
// blocks. Each long block subband then takes a 36-point IMDCT and its window, which is the normal
// window in the long part of a mixed block. Each short block
// subband takes three 12-point IMDCTs with short windows, which start 6, 12 and 18 samples into the
// subband's block.
//
// The first half of each block adds to the second half of the subband's block before, and the
// second half waits for the next granule. Every odd sample of every odd subband is then negated,
// which undoes the frequency inversion of the polyphase filterbank.
//
// Parameters:
//   - dst: the samples, at [sample][subband].
//   - lines: the granule's lines, 18 for each subband. A short block subband holds line f of
//     window w at 3f+w.
//   - block: which subbands hold long blocks, and their windows.
func (f *Filterbank) Transform(dst *[Samples][Subbands]float64, lines *[granuleLines]float64, block Block) {
	reduceAliases(lines, block.Long)

	var out [longPoints]float64

	for subband := range Subbands {
		in := lines[subband*Samples : (subband+1)*Samples]

		switch {
		case subband < block.Mixed:
			longBlock(&out, in, Normal)
		case subband < block.Long:
			longBlock(&out, in, block.Window)
		default:
			shortBlock(&out, in)
		}

		overlap := &f.overlap[subband]

		for at := range Samples {
			sample := out[at] + overlap[at]
			if subband&at&1 == 1 {
				sample = -sample
			}

			dst[at][subband] = sample
			overlap[at] = out[Samples+at]
		}
	}
}
