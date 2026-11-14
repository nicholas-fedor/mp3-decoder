// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package codebook

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxLen is the length of the longest Layer III code word.
const maxLen = 19

// usedBigValues lists the big_values table numbers that have code words.
func usedBigValues() []int {
	var numbers []int

	for number := range BigValueTables {
		if number != 4 && number != 14 {
			numbers = append(numbers, number)
		}
	}

	return numbers
}

// codeWords returns the code words of every table, named for failure messages.
func codeWords(t *testing.T) map[string][]Code {
	t.Helper()

	out := make(map[string][]Code)

	for _, number := range usedBigValues() {
		table, ok := BigValue(number)
		require.True(t, ok)

		codes := make([]Code, 0, table.Width()*table.Width())

		for x := range table.Width() {
			for y := range table.Width() {
				codes = append(codes, table.Code(x, y))
			}
		}

		out[fmt.Sprintf("big_values table %d", number)] = codes
	}

	for number, name := range []string{"A", "B"} {
		table, ok := Count1(number)
		require.True(t, ok)

		codes := make([]Code, 0, Count1Values)

		for value := range Count1Values {
			codes = append(codes, table.Code(value))
		}

		out["count1 table "+name] = codes
	}

	return out
}

// TestKraft checks that each table's code words fill the code space exactly, which a mistyped
// length breaks.
func TestKraft(t *testing.T) {
	t.Parallel()

	for name, codes := range codeWords(t) {
		var sum uint64

		for _, code := range codes {
			require.LessOrEqual(t, code.Len, maxLen, name)
			sum += 1 << (maxLen - code.Len)
		}

		assert.Equal(t, uint64(1)<<maxLen, sum, name)
	}
}

// TestPrefixFree checks that no code word of a table begins another, which a mistyped code word
// breaks.
func TestPrefixFree(t *testing.T) {
	t.Parallel()

	for name, codes := range codeWords(t) {
		words := make([]string, 0, len(codes))
		for _, code := range codes {
			require.Less(t, code.Bits, uint32(1)<<code.Len, name)
			words = append(words, fmt.Sprintf("%0*b", code.Len, code.Bits))
		}

		slices.Sort(words)

		for i := 1; i < len(words); i++ {
			if words[i-1] != "" {
				assert.False(t, strings.HasPrefix(words[i], words[i-1]), "%s: %s begins %s", name, words[i-1], words[i])
			}
		}
	}
}

// TestBigValueShapes checks each big_values table's width and linbits, and that the tables of each
// family share their code words exactly.
func TestBigValueShapes(t *testing.T) {
	t.Parallel()

	widths := []int{1, 2, 3, 3, 0, 4, 4, 6, 6, 6, 8, 8, 8, 16, 0, 16}
	linbits := []int{1, 2, 3, 4, 6, 8, 10, 13, 4, 5, 6, 7, 8, 9, 11, 13}

	for _, number := range usedBigValues() {
		table, _ := BigValue(number)

		if number < 16 {
			assert.Equal(t, widths[number], table.Width(), "table %d", number)
			assert.Zero(t, table.Linbits(), "table %d", number)

			continue
		}

		family, _ := BigValue(number &^ 7)
		if number >= 24 {
			family, _ = BigValue(24)
		}

		assert.Equal(t, 16, table.Width(), "table %d", number)
		assert.Equal(t, linbits[number-16], table.Linbits(), "table %d", number)

		for x := range table.Width() {
			for y := range table.Width() {
				require.Equal(t, family.Code(x, y), table.Code(x, y), "table %d at %d, %d", number, x, y)
			}
		}
	}
}

// TestPublished checks code words against the published tables.
//
// It covers tables 0 and 1 whole, entries at either end of the largest tables, and the first and
// last code words of count1 table A.
func TestPublished(t *testing.T) {
	t.Parallel()

	zero, _ := BigValue(0)
	assert.Equal(t, 1, zero.Width())
	assert.Equal(t, Code{Bits: 0, Len: 0}, zero.Code(0, 0))

	tests := []struct {
		table, x, y int
		want        Code
	}{
		{1, 0, 0, Code{0b1, 1}},
		{1, 0, 1, Code{0b001, 3}},
		{1, 1, 0, Code{0b01, 2}},
		{1, 1, 1, Code{0b000, 3}},
		{13, 0, 0, Code{0b1, 1}},
		{13, 15, 14, Code{0b0000000000000000000, 19}},
		{13, 15, 15, Code{0b0000000000000001, 16}},
		{15, 0, 0, Code{0b111, 3}},
		{16, 0, 0, Code{0b1, 1}},
		{24, 0, 0, Code{0b1111, 4}},
		{24, 15, 15, Code{0b0011, 4}},
	}

	for _, tt := range tests {
		table, _ := BigValue(tt.table)
		assert.Equal(t, tt.want, table.Code(tt.x, tt.y), "table %d at %d, %d", tt.table, tt.x, tt.y)
	}

	a, _ := Count1(0)
	assert.Equal(t, Code{0b1, 1}, a.Code(0))
	assert.Equal(t, Code{0b000001, 6}, a.Code(15))
}

// TestUnused covers the unused tables 4 and 14 and numbers outside the tables.
func TestUnused(t *testing.T) {
	t.Parallel()

	for _, number := range []int{4, 14, -1, BigValueTables} {
		_, ok := BigValue(number)
		assert.False(t, ok, "big_values table %d", number)
	}

	for _, number := range []int{-1, Count1Tables} {
		_, ok := Count1(number)
		assert.False(t, ok, "count1 table %d", number)
	}
}

// TestUnmark covers the empty code word, a short one and the longest.
func TestUnmark(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Code{Bits: 0, Len: 0}, unmark(0b1))
	assert.Equal(t, Code{Bits: 0b0101, Len: 4}, unmark(0b1_0101))
	assert.Equal(t, Code{Bits: 1, Len: 19}, unmark(0b1_0000000000000000001))
}
