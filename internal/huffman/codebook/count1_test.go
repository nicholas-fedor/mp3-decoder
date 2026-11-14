// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package codebook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCount1B checks table B's rule against the published table, which lists the code words of
// the values 0 to 15 from 1111 down to 0000.
func TestCount1B(t *testing.T) {
	t.Parallel()

	b, ok := Count1(1)
	require.True(t, ok)

	for value := range Count1Values {
		want := Code{Bits: uint32(15 - value), Len: 4}

		assert.Equal(t, want, count1B(value), "value %d", value)
		assert.Equal(t, want, b.Code(value), "value %d", value)
	}
}
