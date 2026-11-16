// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rangeServer serves content with range support and counts the requests.
//
// Requests after the first fail with 500 once failAfter is set.
func rangeServer(t *testing.T, content []byte, failAfter bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) > 1 && failAfter {
			http.Error(w, "gone", http.StatusInternalServerError)

			return
		}

		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(content))
	}))
	t.Cleanup(server.Close)

	return server, &requests
}

// counting returns n bytes, each the low byte of its offset.
func counting(n int) []byte {
	data := make([]byte, n)
	for at := range data {
		data[at] = byte(at)
	}

	return data
}

// TestRemoteReadAt reads across block boundaries, up to and past the end, and checks the bytes
// and that a read inside the kept block sends no request.
func TestRemoteReadAt(t *testing.T) {
	t.Parallel()

	content := counting(2*blockSize + blockSize/2)
	server, requests := rangeServer(t, content, false)

	file, err := openRemote(t.Context(), server.Client(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), file.size)

	tests := []struct {
		name     string
		off      int
		size     int
		want     int
		eof      bool
		requests int32
	}{
		{name: "inside the first block", off: 10, size: 100, want: 100, eof: false, requests: 1},
		{name: "across a boundary", off: blockSize - 5, size: 10, want: 10, eof: false, requests: 2},
		{name: "back inside the second block", off: blockSize + 7, size: 3, want: 3, eof: false, requests: 2},
		{name: "past the end", off: len(content) - 4, size: 10, want: 4, eof: true, requests: 3},
		{name: "after the end", off: len(content), size: 1, want: 0, eof: true, requests: 3},
	}

	for _, test := range tests {
		dst := make([]byte, test.size)

		read, err := file.ReadAt(dst, int64(test.off))
		if test.eof {
			require.ErrorIs(t, err, io.EOF, test.name)
		} else {
			require.NoError(t, err, test.name)
		}

		assert.Equal(t, test.want, read, test.name)
		assert.Equal(t, content[test.off:test.off+read], dst[:read], test.name)
		assert.Equal(t, test.requests, requests.Load(), test.name)
	}
}

// TestRemoteReadAtFails checks that a block that fails to arrive is reported with the bytes read
// before it.
func TestRemoteReadAtFails(t *testing.T) {
	t.Parallel()

	server, _ := rangeServer(t, counting(blockSize+10), true)

	file, err := openRemote(t.Context(), server.Client(), server.URL)
	require.NoError(t, err)

	read, err := file.ReadAt(make([]byte, 20), blockSize-10)
	require.ErrorIs(t, err, ErrNoRange)
	assert.Equal(t, 10, read)
}

// TestOpenRemoteErrors checks that a server that ignores the range, answers with an error, sends
// another range or cuts the block short is refused, as are a bad URL and a canceled context.
func TestOpenRemoteErrors(t *testing.T) {
	t.Parallel()

	handlers := map[string]http.HandlerFunc{
		"whole file": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("whole")) },
		"not found":  func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"other range": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Range", "bytes 5-9/10")
			w.WriteHeader(http.StatusPartialContent)
		},
		"short block": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Range", "bytes 0-9/10")
			w.Header().Set("Content-Length", strconv.Itoa(10))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("short"))
		},
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)

			_, err := openRemote(t.Context(), server.Client(), server.URL)
			require.Error(t, err)
		})
	}

	t.Run("bad URL", func(t *testing.T) {
		t.Parallel()

		_, err := openRemote(t.Context(), http.DefaultClient, "http://host/\x7f")
		require.Error(t, err)
	})

	t.Run("canceled", func(t *testing.T) {
		t.Parallel()

		server, _ := rangeServer(t, counting(10), false)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := openRemote(ctx, server.Client(), server.URL)
		require.ErrorIs(t, err, context.Canceled)
	})
}
