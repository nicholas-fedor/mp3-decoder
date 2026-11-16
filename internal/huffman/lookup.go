// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package huffman

import (
	"maps"
	"slices"

	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"
)

//go:generate go test -run ^TestLookupTables$ -update

// lookupRef places one table in the lookup entries.
type lookupRef struct {
	// offset is the index of the table's first entry.
	offset uint16
	// root is the number of bits the table's first level indexes by.
	root uint8
	// linbits is the number of bits after a magnitude of 15 that extend it.
	linbits uint8
}

// lookups holds the decode tables of every Huffman table.
type lookups struct {
	// entries holds every table's entries, one table after another.
	entries []uint16
	// bigValues places each big_values table at its number.
	bigValues [codebook.BigValueTables]lookupRef
	// count1A places count1 table A.
	count1A lookupRef
}

// codeWord is one code word with the values it codes.
type codeWord struct {
	// code is the code word.
	code codebook.Code
	// x is the first value, or 0 for a count1 code word.
	x int
	// y is the second value, or the quadruple of a count1 code word.
	y int
}

// Lookup entry layout.
//
// A leaf entry holds the code word's values and the number of its bits that the entry's level
// takes. A pointer entry holds where a subtable starts, relative to its table, and the number of
// bits the subtable indexes by.
const (
	// pointer marks a pointer entry.
	pointer = 1 << 15
	// subShift is the shift of a pointer's subtable width.
	subShift = 11
	// subMask keeps a pointer's subtable width.
	subMask = 0xf
	// offsetMask keeps a pointer's subtable offset.
	offsetMask = 1<<subShift - 1
	// lenShift is the shift of a leaf's bit count.
	lenShift = 8
	// lenMask keeps a leaf's bit count.
	lenMask = 0xf
	// xShift is the shift of a leaf's first value.
	xShift = 4
	// valueMask keeps one of a leaf's values.
	valueMask = 0xf
)

// Lookup table sizes.
const (
	// lookupSize is the number of entries, a power of two at least as large as every table
	// together, so an index masked to that size stays in range.
	lookupSize = 1 << 13
	// lookupMask keeps an index within the entries.
	lookupMask = lookupSize - 1
	// maxTable is the most entries one table takes, which a pointer's offset can reach.
	maxTable = 1 << subShift
	// maxLen is the length of the longest code word.
	maxLen = 19
	// subtableShare is how rarely a decode reaches a subtable, as at most 1 in subtableShare.
	//
	// It is measured by the probability of 2^-length that a code word's length implies.
	subtableShare = 20
)

// readCode reads one code word through its table's lookup entries.
//
// Parameters:
//   - entries: the lookup entries.
//   - ref: where the table's entries are.
//   - stream: the bitstream, at the code word.
//
// Returns:
//   - x: the first value, or 0 for a count1 code word.
//   - y: the second value, or the quadruple of a count1 code word.
//
//nolint:nonamedreturns // Same-type returns need names.
func readCode(entries *[lookupSize]uint16, ref lookupRef, stream *bits.Reader) (x, y int) {
	entry := entries[(int(ref.offset)+int(stream.Peek(int(ref.root))))&lookupMask]
	if entry&pointer != 0 {
		stream.Skip(int(ref.root))

		width := int(entry >> subShift & subMask)

		entry = entries[(int(ref.offset)+int(entry&offsetMask)+int(stream.Peek(width)))&lookupMask]
	}

	stream.Skip(int(entry >> lenShift & lenMask))

	return int(entry >> xShift & valueMask), int(entry & valueMask)
}

// buildLookups builds the decode tables of every Huffman table from its code words.
//
// Tables that share code words share entries. The unused tables 4 and 14 decode as table 0, whose
// only value is x and y of 0, coded in no bits.
//
// Returns:
//   - lookups: the tables.
func buildLookups() lookups {
	var built lookups

	for number := range codebook.BigValueTables {
		table, ok := codebook.BigValue(number)
		if !ok {
			table, _ = codebook.BigValue(0)
		}

		ref := built.add(bigValueWords(table))

		ref.linbits = uint8(table.Linbits()) //nolint:gosec // At most 13.
		built.bigValues[number] = ref
	}

	a, _ := codebook.Count1(0)

	built.count1A = built.add(count1Words(a))

	return built
}

// add adds a table's entries, or finds the same entries already added.
//
// Parameters:
//   - words: the table's code words.
//
// Returns:
//   - lookupRef: where the table's entries are, without linbits.
func (l *lookups) add(words []codeWord) lookupRef {
	root := chooseRoot(words)
	entries := tableEntries(words, root)

	for at := 0; at+len(entries) <= len(l.entries); at++ {
		if slices.Equal(l.entries[at:at+len(entries)], entries) {
			return lookupRef{offset: uint16(at), root: uint8(root), linbits: 0} //nolint:gosec // Within the entries.
		}
	}

	offset := len(l.entries)

	l.entries = append(l.entries, entries...)

	return lookupRef{offset: uint16(offset), root: uint8(root), linbits: 0} //nolint:gosec // Within the entries.
}

// chooseRoot chooses how many bits a table's first level indexes by.
//
// It is the fewest bits that send at most 1 decode in 20 to a subtable, with every subtable narrow
// enough for a pointer to give its width and the table within the offsets a pointer reaches.
//
// Parameters:
//   - words: the table's code words.
//
// Returns:
//   - int: the number of bits.
func chooseRoot(words []codeWord) int {
	root := 0
	for !meetsRule(words, root) {
		root++
	}

	return root
}

// meetsRule reports whether a first level of root bits keeps a table within its rule.
//
// At most 1 decode in 20 may reach a subtable, every subtable's width must fit a pointer, and the
// table must fit the offsets a pointer reaches. A first level as wide as the longest code word
// always meets it.
//
// Parameters:
//   - words: the table's code words.
//   - root: the number of bits the first level indexes by.
//
// Returns:
//   - bool: whether the rule holds.
func meetsRule(words []codeWord, root int) bool {
	var deep uint64

	longest := 0

	for _, word := range words {
		longest = max(longest, word.code.Len)

		if word.code.Len > root {
			deep += 1 << (maxLen - word.code.Len)
		}
	}

	return deep*subtableShare <= 1<<maxLen && longest-root <= subMask && len(tableEntries(words, root)) <= maxTable
}

// tableEntries lays out one table's entries.
//
// The entries are a first level indexed by root bits, then a subtable for each prefix whose code
// words are longer.
//
// Parameters:
//   - words: the table's code words.
//   - root: the number of bits the first level indexes by.
//
// Returns:
//   - []uint16: the entries.
func tableEntries(words []codeWord, root int) []uint16 {
	widths := map[uint32]int{}

	for _, word := range words {
		if word.code.Len > root {
			prefix := word.code.Bits >> (word.code.Len - root)

			widths[prefix] = max(widths[prefix], word.code.Len-root)
		}
	}

	size := 1 << root
	for _, width := range widths {
		size += 1 << width
	}

	entries := make([]uint16, size) //nolint:makezero // Entries are set by index.
	subtables := map[uint32]int{}
	next := 1 << root

	for _, prefix := range slices.Sorted(maps.Keys(widths)) {
		subtables[prefix] = next
		entries[prefix] = uint16(pointer | widths[prefix]<<subShift | next) //nolint:gosec // Fits the layout.

		next += 1 << widths[prefix]
	}

	for _, word := range words {
		if word.code.Len <= root {
			fill(entries, word.code.Bits<<(root-word.code.Len), 1<<(root-word.code.Len), leaf(word.code.Len, word))

			continue
		}

		prefix := word.code.Bits >> (word.code.Len - root)
		rest := word.code.Len - root
		width := widths[prefix]
		at := uint32(subtables[prefix]) + (word.code.Bits&(1<<rest-1))<<(width-rest) //nolint:gosec // Small.
		fill(entries, at, 1<<(width-rest), leaf(rest, word))
	}

	return entries
}

// leaf returns a leaf entry.
//
// Parameters:
//   - length: the number of the code word's bits the entry's level takes.
//   - word: the code word.
//
// Returns:
//   - uint16: the entry.
func leaf(length int, word codeWord) uint16 {
	return uint16(length<<lenShift | word.x<<xShift | word.y) //nolint:gosec // Fits the layout.
}

// fill sets a run of entries.
//
// Parameters:
//   - entries: the entries.
//   - at: the first entry of the run.
//   - n: the number of entries.
//   - entry: the entry to set.
func fill(entries []uint16, at uint32, n int, entry uint16) {
	for i := range n {
		entries[int(at)+i] = entry
	}
}

// bigValueWords lists a big_values table's code words.
//
// Parameters:
//   - table: the table.
//
// Returns:
//   - []codeWord: the code words with their pairs.
func bigValueWords(table codebook.BigValueTable) []codeWord {
	words := make([]codeWord, 0, table.Width()*table.Width())

	for x := range table.Width() {
		for y := range table.Width() {
			words = append(words, codeWord{code: table.Code(x, y), x: x, y: y})
		}
	}

	return words
}

// count1Words lists a count1 table's code words.
//
// Parameters:
//   - table: the table.
//
// Returns:
//   - []codeWord: the code words with their quadruples.
func count1Words(table codebook.Count1Table) []codeWord {
	words := make([]codeWord, 0, codebook.Count1Values)

	for value := range codebook.Count1Values {
		words = append(words, codeWord{code: table.Code(value), x: 0, y: value})
	}

	return words
}
