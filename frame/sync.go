// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package frame

// verdict is what the bytes after a frame say about it.
type verdict uint8

// Verdicts.
const (
	// rejected means the bytes after the frame do not hold a header of its stream.
	rejected verdict = iota + 1
	// confirmed means a header of the frame's stream follows it.
	confirmed
	// undecided means the bytes end before they can confirm or reject the frame.
	undecided
)

// Find finds the first frame in window that the header after it confirms.
//
// A frame counts only when accept takes its header. A free-format frame needs two headers after it
// at one spacing, and its returned header carries the bitrate that spacing gives. The search stops
// at the first frame that the window ends before confirming, since a longer window may confirm it.
//
// Parameters:
//   - window: the bytes to search.
//   - accept: reports whether a header may start the frame, or nil to take any header.
//
// Returns:
//   - int: the offset of the confirmed frame, or else the offset where a search of a longer window
//     starts again.
//   - Header: the header at that offset, or the zero Header when there is none.
//   - bool: whether the frame was confirmed.
func Find(window []byte, accept func(Header) bool) (int, Header, bool) {
	var none Header

	for at := 0; at+HeaderSize <= len(window); at++ {
		head, ok := Parse(window[at:])
		if !ok || (accept != nil && !accept(head)) {
			continue
		}

		head, judged := judge(window[at:], head)

		switch judged {
		case confirmed:
			return at, head, true
		case undecided:
			return at, head, false
		default:
		}
	}

	// A header needs all its bytes, so a longer window looks at the last few offsets again.
	return max(0, len(window)-HeaderSize+1), none, false
}

// judge judges the frame at the start of frame by the headers after it.
//
// Parameters:
//   - frame: the bytes from the frame's header onward.
//   - head: the frame's header.
//
// Returns:
//   - Header: the header, with the measured bitrate of a confirmed free-format frame.
//   - verdict: the verdict.
func judge(frame []byte, head Header) (Header, verdict) {
	if head.Free {
		return measure(frame, head)
	}

	return head, follows(frame, head, head.Size())
}

// measure measures a free-format frame by the two headers after it.
//
// The frame's size without padding is the spacing of every frame in the stream, so the header
// after the next one must lie at that spacing plus the next frame's padding. The spacing is at
// least the size of the header, CRC and side information.
//
// Parameters:
//   - frame: the bytes from the frame's header onward.
//   - head: the frame's header.
//
// Returns:
//   - Header: the header, with the bitrate the frame's size gives when it is confirmed.
//   - verdict: the verdict.
func measure(frame []byte, head Header) (Header, verdict) {
	for size := head.DataAt() + head.padding(); size <= MaxSize; size++ {
		switch follows(frame, head, size) {
		case undecided:
			return head, undecided
		case confirmed:
		default:
			continue
		}

		spacing := size - head.padding()
		if spacing%head.slot() != 0 {
			continue
		}

		next, _ := Parse(frame[size:])

		switch follows(frame, head, size+spacing+next.padding()) {
		case confirmed:
			head.Bitrate = head.freeBitrate(spacing)

			return head, confirmed
		case undecided:
			return head, undecided
		default:
		}
	}

	return head, rejected
}

// follows judges whether a header of head's stream starts at offset at of frame.
//
// Parameters:
//   - frame: the bytes from the frame's header onward.
//   - head: the frame's header.
//   - at: the offset of the next header.
//
// Returns:
//   - verdict: the verdict.
func follows(frame []byte, head Header, at int) verdict {
	if at+HeaderSize > len(frame) {
		return undecided
	}

	next, ok := Parse(frame[at:])
	if ok && head.SameStream(next) {
		return confirmed
	}

	return rejected
}

// freeBitrate returns the bitrate of a free-format stream whose unpadded frames hold spacing bytes.
//
// It rounds up, so Size gives spacing back.
//
// Parameters:
//   - spacing: the size of an unpadded frame in bytes.
//
// Returns:
//   - int: the bitrate in bit/s.
func (h Header) freeBitrate(spacing int) int {
	bits := spacing * bitsPerByte * h.SampleRate

	return (bits + h.Samples() - 1) / h.Samples()
}
