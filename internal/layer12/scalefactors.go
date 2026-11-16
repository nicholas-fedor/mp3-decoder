// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

//go:generate go test -run ^TestScalefactorTable$ -update

// scalefactorIndices is the number of scalefactor indices, of which the last is not used.
const scalefactorIndices = 64
