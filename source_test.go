// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package mp3

import (
	"bytes"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSourceEnsure checks that ensure keeps the unused bytes in order across reads of a byte at a
// time, and that it stops at the end of the stream.
func TestSourceEnsure(t *testing.T) {
	t.Parallel()

	data := make([]byte, 3*readSize)
	for at := range data {
		data[at] = byte(at % 251)
	}

	s := source{reader: iotest.OneByteReader(bytes.NewReader(data))}

	require.NoError(t, s.ensure(100))
	assert.Equal(t, data[:100], s.bytes())

	s.consume(60)
	require.NoError(t, s.ensure(2*readSize))
	assert.Equal(t, data[60:60+2*readSize], s.bytes())

	require.NoError(t, s.ensure(len(data)))
	assert.Equal(t, data[60:], s.bytes())
	assert.True(t, s.ended)
}

// TestSourceDiscard checks that discard moves past buffered bytes, then reads past the rest.
func TestSourceDiscard(t *testing.T) {
	t.Parallel()

	data := make([]byte, 4*readSize)
	for at := range data {
		data[at] = byte(at % 251)
	}

	s := source{reader: bytes.NewReader(data)}

	require.NoError(t, s.ensure(10))
	require.NoError(t, s.discard(2*readSize+5))
	require.NoError(t, s.ensure(10))
	assert.Equal(t, data[2*readSize+5:2*readSize+15], s.bytes()[:10])

	require.NoError(t, s.discard(len(data)))
	assert.True(t, s.ended)
	assert.Empty(t, s.bytes())
}
