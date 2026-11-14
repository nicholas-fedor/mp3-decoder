// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package codebook

import "math/bits"

// Code is one Huffman code word.
type Code struct {
	// Bits is the code word, right-aligned.
	Bits uint32
	// Len is the code word's length in bits.
	Len int
}

// BigValueTable is a Huffman table of the big_values region, which codes pairs of magnitudes x and
// y.
type BigValueTable struct {
	// codes holds the code words in rows by x, each marked by a 1 bit above its highest bit.
	codes [][]uint32
	// linbits is the number of bits after a magnitude of 15 that extend it.
	linbits int
}

// Count1Table is a Huffman table of the count1 region, which codes quadruples of magnitudes v, w, x
// and y of 0 or 1.
type Count1Table struct {
	// codes holds the code words of table A, each marked by a 1 bit above its highest bit, or is nil
	// for table B, whose code words are their values' bits inverted.
	codes *[Count1Values]uint32
}

// Table counts.
const (
	// BigValueTables is the number of big_values table numbers that table_select can give.
	BigValueTables = 32
	// Count1Tables is the number of count1 tables that count1table_select can give.
	Count1Tables = 2
	// Count1Values is the number of quadruples a count1 table codes.
	Count1Values = 16
)

// bigValueTables holds each big_values table at the number table_select gives it.
//
// Tables 4 and 14 are unused. Tables 16 to 23 share one set of code words, and so do tables 24 to
// 31. Each table of a family extends a magnitude of 15 by its own number of linbits.
var bigValueTables = [BigValueTables]BigValueTable{
	0:  {table0, 0},
	1:  {table1, 0},
	2:  {table2, 0},
	3:  {table3, 0},
	5:  {table5, 0},
	6:  {table6, 0},
	7:  {table7, 0},
	8:  {table8, 0},
	9:  {table9, 0},
	10: {table10, 0},
	11: {table11, 0},
	12: {table12, 0},
	13: {table13, 0},
	15: {table15, 0},
	16: {table16, 1},
	17: {table16, 2},
	18: {table16, 3},
	19: {table16, 4},
	20: {table16, 6},
	21: {table16, 8},
	22: {table16, 10},
	23: {table16, 13},
	24: {table24, 4},
	25: {table24, 5},
	26: {table24, 6},
	27: {table24, 7},
	28: {table24, 8},
	29: {table24, 9},
	30: {table24, 11},
	31: {table24, 13},
}

// count1Tables holds count1 tables A and B at the numbers count1table_select gives them.
var count1Tables = [Count1Tables]Count1Table{{&count1A}, {nil}}

// BigValue returns the big_values table that table_select names.
//
// Parameters:
//   - table: the table number.
//
// Returns:
//   - BigValueTable: the table.
//   - bool: false for the unused tables 4 and 14 and for numbers outside 0 to 31.
func BigValue(table int) (BigValueTable, bool) {
	if table < 0 || table >= BigValueTables || bigValueTables[table].codes == nil {
		var none BigValueTable

		return none, false
	}

	return bigValueTables[table], true
}

// Count1 returns the count1 table that count1table_select names.
//
// Parameters:
//   - table: the table number, 0 for table A and 1 for table B.
//
// Returns:
//   - Count1Table: the table.
//   - bool: false for any other number.
func Count1(table int) (Count1Table, bool) {
	if table < 0 || table >= Count1Tables {
		var none Count1Table

		return none, false
	}

	return count1Tables[table], true
}

// Code returns the code word of a pair of magnitudes.
//
// Parameters:
//   - x: the first magnitude, from 0 to Width() - 1.
//   - y: the second magnitude, from 0 to Width() - 1.
//
// Returns:
//   - Code: the code word.
func (t BigValueTable) Code(x, y int) Code {
	return unmark(t.codes[x][y])
}

// Linbits returns the number of bits after a magnitude of 15 that extend it.
//
// Returns:
//   - int: the number of linbits, or 0 for a table without them.
func (t BigValueTable) Linbits() int {
	return t.linbits
}

// Width returns the number of magnitudes x and y each take.
//
// Returns:
//   - int: the width, from 1 for table 0 to 16.
func (t BigValueTable) Width() int {
	return len(t.codes)
}

// Code returns the code word of a quadruple of magnitudes.
//
// Parameters:
//   - value: the quadruple, v<<3 | w<<2 | x<<1 | y.
//
// Returns:
//   - Code: the code word.
func (t Count1Table) Code(value int) Code {
	if t.codes == nil {
		return count1B(value)
	}

	return unmark(t.codes[value])
}

// unmark splits a marked code word into its bits and length.
//
// Parameters:
//   - word: the code word, marked by a 1 bit above its highest bit.
//
// Returns:
//   - Code: the code word without its mark.
func unmark(word uint32) Code {
	length := bits.Len32(word) - 1

	return Code{Bits: word &^ (1 << length), Len: length}
}
