// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package huffman

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/generated"
	"github.com/nicholas-fedor/mp3-decoder/internal/huffman/codebook"
)

// update rewrites lookup_gen.go from the code words instead of checking it.
var update = flag.Bool("update", false, "rewrite lookup_gen.go from the code words")

// generatedFile is the file the lookup tables are generated into.
const generatedFile = "lookup_gen.go"

// TestLookupTables checks that the generated lookup tables are the ones the code words give, or
// rewrites them with -update.
func TestLookupTables(t *testing.T) {
	built := buildLookups()
	require.LessOrEqual(t, len(built.entries), lookupSize)

	if *update {
		require.NoError(t, os.WriteFile(generatedFile, lookupSource(t, built), 0o644))

		return
	}

	t.Parallel()

	assert.Equal(t, built.entries, lookupEntries[:len(built.entries)])
	assert.Equal(t, make([]uint16, lookupSize-len(built.entries)), lookupEntries[len(built.entries):])
	assert.Equal(t, built.bigValues, bigValueLookups)
	assert.Equal(t, built.count1A, count1ALookup)
}

// lookupSource writes the generated file for built lookup tables.
func lookupSource(t *testing.T, built lookups) []byte {
	t.Helper()

	var b bytes.Buffer

	b.WriteString("// lookupEntries holds the decode tables of every Huffman table, one after another.\n")
	b.WriteString("var lookupEntries = [lookupSize]uint16{\n")

	for at := 0; at < len(built.entries); at += 12 {
		for _, entry := range built.entries[at:min(at+12, len(built.entries))] {
			fmt.Fprintf(&b, "0x%04x, ", entry)
		}

		b.WriteString("\n")
	}

	b.WriteString("}\n\n")
	b.WriteString("// bigValueLookups places each big_values table's decode table at its number.\n")
	b.WriteString("var bigValueLookups = [32]lookupRef{\n")

	for number, ref := range built.bigValues {
		fmt.Fprintf(&b, "%d: {%d, %d, %d},\n", number, ref.offset, ref.root, ref.linbits)
	}

	b.WriteString("}\n\n")
	b.WriteString("// count1ALookup places count1 table A's decode table.\n")
	fmt.Fprintf(&b, "var count1ALookup = lookupRef{%d, %d, %d}\n", built.count1A.offset, built.count1A.root, built.count1A.linbits)

	return generated.Source(t, "huffman", b.Bytes())
}

// TestChooseRoot checks that each table's first level meets its rule and that one bit fewer would
// not.
func TestChooseRoot(t *testing.T) {
	t.Parallel()

	built := buildLookups()
	tables := map[string]struct {
		words []codeWord
		ref   lookupRef
	}{}

	for number, ref := range built.bigValues {
		table, ok := codebook.BigValue(number)
		if ok {
			tables[fmt.Sprintf("big_values table %d", number)] = struct {
				words []codeWord
				ref   lookupRef
			}{bigValueWords(table), ref}
		}
	}

	a, _ := codebook.Count1(0)
	tables["count1 table A"] = struct {
		words []codeWord
		ref   lookupRef
	}{count1Words(a), built.count1A}

	for name, table := range tables {
		root := int(table.ref.root)

		assert.True(t, meetsRule(table.words, root), name)

		if root > 0 {
			assert.False(t, meetsRule(table.words, root-1), "%s: %d bits would do", name, root-1)
		}
	}
}

// TestEntriesLayout checks the generated entries themselves: each table fits the offsets a pointer
// reaches, and each pointer's subtable lies within its table.
func TestEntriesLayout(t *testing.T) {
	t.Parallel()

	built := buildLookups()
	starts := []int{int(built.count1A.offset), len(built.entries)}

	for _, ref := range built.bigValues {
		starts = append(starts, int(ref.offset))
	}

	slices.Sort(starts)
	starts = slices.Compact(starts)

	for i := range len(starts) - 1 {
		table := built.entries[starts[i]:starts[i+1]]
		assert.LessOrEqual(t, len(table), maxTable, "table at %d", starts[i])

		for _, entry := range table {
			if entry&pointer != 0 {
				width := int(entry >> subShift & subMask)
				assert.LessOrEqual(t, int(entry&offsetMask)+1<<width, len(table), "table at %d", starts[i])
			}
		}
	}
}
