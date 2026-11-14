// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

// bands is the partition of a granule's lines into scalefactor bands at one sampling frequency.
type bands struct {
	// rate is the sampling frequency in Hz.
	rate int
	// long holds the first line of each long block band, then 576.
	long [longBands + 1]int
	// short holds the first line of each short block band within one window, then 192.
	short [shortBands + 1]int
}

// Short band counts.
const (
	// shortBands is the number of short block bands.
	shortBands = 13
	// mixedShortBand is the first short band of mixed blocks, whose start ends their long bands.
	mixedShortBand = 3
)

// bandTables holds the bands of each sampling frequency, each entry its rate, long bands and short
// bands.
//
// The MPEG-2.5 frequencies 11.025 and 12 kHz share the bands of 16 kHz.
var bandTables = [...]bands{
	{
		44100,
		[...]int{
			0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 52, 62, 74, 90, 110, 134, 162, 196, 238, 288, 342, 418, 576,
		},
		[...]int{0, 4, 8, 12, 16, 22, 30, 40, 52, 66, 84, 106, 136, 192},
	},
	{
		48000,
		[...]int{
			0, 4, 8, 12, 16, 20, 24, 30, 36, 42, 50, 60, 72, 88, 106, 128, 156, 190, 230, 276, 330, 384, 576,
		},
		[...]int{0, 4, 8, 12, 16, 22, 28, 38, 50, 64, 80, 100, 126, 192},
	},
	{
		32000,
		[...]int{
			0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 54, 66, 82, 102, 126, 156, 194, 240, 296, 364, 448, 550, 576,
		},
		[...]int{0, 4, 8, 12, 16, 22, 30, 42, 58, 78, 104, 138, 180, 192},
	},
	{
		22050,
		[...]int{
			0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576,
		},
		[...]int{0, 4, 8, 12, 18, 24, 32, 42, 56, 74, 100, 132, 174, 192},
	},
	{
		24000,
		[...]int{
			0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 114, 136, 162, 194, 232, 278, 332, 394, 464, 540, 576,
		},
		[...]int{0, 4, 8, 12, 18, 26, 36, 48, 62, 80, 104, 136, 180, 192},
	},
	{
		16000,
		[...]int{
			0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576,
		},
		[...]int{0, 4, 8, 12, 18, 26, 36, 48, 62, 80, 104, 134, 174, 192},
	},
	{
		11025,
		[...]int{
			0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576,
		},
		[...]int{0, 4, 8, 12, 18, 26, 36, 48, 62, 80, 104, 134, 174, 192},
	},
	{
		12000,
		[...]int{
			0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576,
		},
		[...]int{0, 4, 8, 12, 18, 26, 36, 48, 62, 80, 104, 134, 174, 192},
	},
	{
		8000,
		[...]int{
			0, 12, 24, 36, 48, 60, 72, 88, 108, 132, 160, 192, 232, 280, 336, 400, 476, 566, 568, 570, 572, 574, 576,
		},
		[...]int{0, 8, 16, 24, 36, 52, 72, 96, 124, 160, 162, 164, 166, 192},
	},
}

// bandsAt returns the bands of a sampling frequency.
//
// Parameters:
//   - rate: the sampling frequency in Hz, one a frame header gives.
//
// Returns:
//   - *bands: the bands, or nil for a frequency no frame header gives.
func bandsAt(rate int) *bands {
	for at := range bandTables {
		if bandTables[at].rate == rate {
			return &bandTables[at]
		}
	}

	return nil
}
