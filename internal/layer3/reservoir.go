// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import "github.com/nicholas-fedor/mp3-decoder/frame"

// Reservoir is the bit reservoir, the main data of past frames that a frame's main data may begin
// in.
//
// It holds the last 511 bytes, the furthest any main_data_begin reaches back. The zero Reservoir is
// empty and ready to use.
type Reservoir struct {
	// data holds the reservoir the last frame began in, followed by that frame's own main data.
	data [reservoirSize + frame.MaxSize]byte
	// filled is how many bytes at the start of data the last frame filled.
	filled int
}

// reservoirSize is the most bytes the reservoir holds, the largest 9-bit main_data_begin.
const reservoirSize = 1<<mpeg1Begin - 1

// MainData returns a frame's main data and adds the frame's own main data to the reservoir.
//
// When the reservoir holds fewer bytes than begin, as at the start of a stream joined midway or
// after a resync, the main data starts at the first byte it holds. The frame's own main data is
// kept either way, so the frames after it can begin in it. Bytes of own past [frame.MaxSize] belong
// to no frame and are dropped.
//
// Parameters:
//   - begin: the frame's main_data_begin.
//   - own: the frame's bytes after its side information, to the end of the frame.
//
// Returns:
//   - []byte: the main data, from begin bytes before own to the end of own, valid until the next
//     call to MainData or Reset.
//   - int: how many of the begin bytes the reservoir lacks.
func (r *Reservoir) MainData(begin int, own []byte) ([]byte, int) {
	held := min(r.filled, reservoirSize)
	copy(r.data[:], r.data[r.filled-held:r.filled])

	r.filled = held + copy(r.data[held:], own[:min(len(own), frame.MaxSize)])

	reach := min(begin, held)

	return r.data[held-reach : r.filled], begin - reach
}

// Reset empties the reservoir, as a resync or a seek needs, since the frames before it no longer
// lead to the next.
func (r *Reservoir) Reset() {
	r.filled = 0
}
