// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3

// frameIndex holds where the search for every step-th frame of a stream starts.
//
// Each entry is the stream's position just after the frame before, from which the scanner finds
// the frame as it did the first time. When the index fills, it drops every other entry and doubles
// its step. It thus stays small for a stream of any length. Its zero value is empty, with a step of
// one frame.
type frameIndex struct {
	// offsets holds the position from which the search for frame i<<shift starts, at i.
	offsets []int64
	// shift is the base-2 logarithm of the step.
	shift uint
}

// indexSize is the most entries the index holds.
const indexSize = 1024

// add records where the search for a frame started, when the frame is the next one the index
// takes.
//
// Parameters:
//   - frame: the frame's number, from 0 for the first audio frame.
//   - offset: the position in the stream where the search for it started.
func (x *frameIndex) add(frame, offset int64) {
	if frame != int64(len(x.offsets))<<x.shift {
		return
	}

	if len(x.offsets) == indexSize {
		for at := range indexSize / 2 {
			x.offsets[at] = x.offsets[2*at]
		}

		// The frame is the first of the doubled step after the entries kept.
		x.offsets = x.offsets[:indexSize/2]
		x.shift++
	}

	if x.offsets == nil {
		x.offsets = make([]int64, 0, indexSize)
	}

	x.offsets = append(x.offsets, offset)
}

// before returns the last entry at or before a frame.
//
// Parameters:
//   - frame: the frame's number.
//
// Returns:
//   - from: the number of the entry's frame.
//   - offset: the position where the search for that frame starts.
//   - ok: whether the index holds an entry at or before the frame.
//
//nolint:nonamedreturns // Same-type returns need names.
func (x *frameIndex) before(frame int64) (from, offset int64, ok bool) {
	if len(x.offsets) == 0 || frame < 0 {
		return 0, 0, false
	}

	at := min(frame>>x.shift, int64(len(x.offsets)-1))

	return at << x.shift, x.offsets[at], true
}
