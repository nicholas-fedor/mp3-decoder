// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package huffman decodes the Huffman-coded spectral values of a Layer III granule.
//
// [BigValues] decodes the pairs of the big_values regions, with their linbits and signs. [Count1]
// decodes the quadruples of the count1 region. Both decode through generated multi-level lookup
// tables, which the code tables of package codebook define.
package huffman
