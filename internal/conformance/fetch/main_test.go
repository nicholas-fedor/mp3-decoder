// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunUsage checks that a missing -dir, an unknown flag and an argument after the flags exit
// with 2.
func TestRunUsage(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"-unknown"}, {"-dir", t.TempDir(), "extra"}} {
		var stdout, stderr bytes.Buffer

		assert.Equal(t, exitUsage, run(t.Context(), args, &stdout, &stderr), "%+v", args)
		assert.Empty(t, stdout.String(), "%+v", args)
		assert.NotEmpty(t, stderr.String(), "%+v", args)
	}
}

// TestRunFails checks that a fetch from a server without the archives exits with 1.
func TestRunFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	var stdout, stderr bytes.Buffer

	code := run(t.Context(), []string{"-dir", t.TempDir(), "-url", server.URL}, &stdout, &stderr)
	assert.Equal(t, exitFailure, code)
	assert.Contains(t, stderr.String(), "fetch: ")
	assert.Empty(t, stdout.String())
}

// TestRunReady checks that a complete cache, which MP3_DECODER_CONFORMANCE names, is ready without a
// request: the URL leads nowhere.
func TestRunReady(t *testing.T) {
	t.Parallel()

	dir := os.Getenv("MP3_DECODER_CONFORMANCE")
	if dir == "" {
		t.Skip("MP3_DECODER_CONFORMANCE names no cache of conformance streams")
	}

	var stdout, stderr bytes.Buffer

	code := run(t.Context(), []string{"-dir", dir, "-url", "http://127.0.0.1:0/"}, &stdout, &stderr)
	assert.Equal(t, 0, code, stderr.String())
	assert.Equal(t, "conformance streams ready in "+dir+", 0 fetched\n", stdout.String())
}
