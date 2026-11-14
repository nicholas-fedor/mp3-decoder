// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
)

// step is one frame fed to a reservoir, with the main data and missing count it must get.
type step struct {
	// main is the main data the frame must get.
	main string
	// own is the frame's own main data.
	own []byte
	// begin is the frame's main_data_begin.
	begin int
	// missing is the missing count the frame must get.
	missing int
	// reset reports whether the reservoir is reset before the frame.
	reset bool
}

// TestMainDataSequences feeds the reservoir scripted frames and checks the main data and missing
// count each frame gets, from the start of a stream, across frames, over the 511 bytes it holds,
// and after a resync.
func TestMainDataSequences(t *testing.T) {
	t.Parallel()

	long := counting(600)
	huge := counting(frame.MaxSize + 10)

	tests := []struct {
		name  string
		steps []step
	}{
		{"stream start", []step{{own: []byte("abc"), main: "abc"}}},
		{"empty frame", []step{{main: ""}}},
		{"begins in the frame before", []step{
			{own: []byte("abcd"), main: "abcd"},
			{begin: 2, own: []byte("ef"), main: "cdef"},
		}},
		{"reaches back across frames", []step{
			{own: []byte("ab"), main: "ab"},
			{own: []byte("cd"), main: "cd"},
			{begin: 3, own: []byte("e"), main: "bcde"},
		}},
		{"underflow at the start", []step{{begin: 5, own: []byte("abc"), main: "abc", missing: 5}}},
		{"partial underflow", []step{
			{own: []byte("ab"), main: "ab"},
			{begin: 5, own: []byte("cd"), main: "abcd", missing: 3},
		}},
		{"underflow keeps the frame's main data", []step{
			{begin: 5, own: []byte("abc"), main: "abc", missing: 5},
			{begin: 3, own: []byte("d"), main: "abcd"},
		}},
		{"underflow after a resync", []step{
			{own: []byte("abcdef"), main: "abcdef"},
			{reset: true, begin: 2, own: []byte("gh"), main: "gh", missing: 2},
			{begin: 3, own: []byte("i"), main: "ghi", missing: 1},
		}},
		{"holds the last 511 bytes", []step{
			{own: long[:300], main: string(long[:300])},
			{own: long[300:], main: string(long[300:])},
			{begin: 511, own: []byte("x"), main: string(long[89:]) + "x"},
			{begin: 511, own: nil, main: string(long[90:]) + "x"},
		}},
		{"drops bytes past the largest frame", []step{
			{own: huge, main: string(huge[:frame.MaxSize])},
			{begin: 511, own: []byte("x"), main: string(huge[frame.MaxSize-511:frame.MaxSize]) + "x"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var reservoir layer3.Reservoir

			for i, step := range tt.steps {
				if step.reset {
					reservoir.Reset()
				}

				main, missing := reservoir.MainData(step.begin, step.own)

				assert.Equal(t, step.main, string(main), "step %d", i)
				assert.Equal(t, step.missing, missing, "step %d", i)
			}
		})
	}
}

// TestMainDataEncodedFiles walks files LAME encoded and checks that every frame's main data is
// whole and holds its granules.
//
// It then joins each file at its first frame that begins in the reservoir. Each frame of the join
// gets the end of the main data the whole file gives it, and missing counts the bytes it lacks.
// After the first frames the join's main data matches the whole file's.
func TestMainDataEncodedFiles(t *testing.T) {
	t.Parallel()

	for _, name := range encodedFiles {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			frames := encodedFrames(t, name)

			var (
				side  layer3.SideInfo
				whole layer3.Reservoir
				mains [][]byte
			)

			join := 0

			for at, data := range frames {
				head, _ := frame.Parse(data)
				require.NoError(t, side.Parse(head, data))

				main, missing := whole.MainData(side.MainDataBegin, data[head.DataAt():])
				require.Zero(t, missing, "frame %d", at)
				assert.LessOrEqual(t, granuleBits(head, &side), 8*len(main), "frame %d", at)

				mains = append(mains, slices.Clone(main))

				if join == 0 && at > 0 && side.MainDataBegin > 0 {
					join = at
				}
			}

			require.Positive(t, join)

			var joined layer3.Reservoir

			missed := 0

			for at, data := range frames[join:] {
				head, _ := frame.Parse(data)
				require.NoError(t, side.Parse(head, data))

				main, missing := joined.MainData(side.MainDataBegin, data[head.DataAt():])
				want := mains[join+at]

				assert.Equal(t, want[missing:], main, "frame %d", join+at)

				if missing > 0 {
					missed++
				}
			}

			assert.Positive(t, missed)
			assert.Less(t, missed, len(frames)-join)
		})
	}
}

// counting returns n bytes that count up from 0, wrapping at 256.
func counting(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i)
	}

	return data
}
