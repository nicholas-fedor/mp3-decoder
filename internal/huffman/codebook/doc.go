// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

// Package codebook holds the Layer III Huffman code tables.
//
// The big_values region of a granule codes pairs of magnitudes with the table that table_select
// names. The count1 region codes quadruples of magnitudes of 0 or 1 with table A or B, which
// count1table_select names. Signs, and the linbits that extend a magnitude of 15, follow each code
// word outside the code.
package codebook
