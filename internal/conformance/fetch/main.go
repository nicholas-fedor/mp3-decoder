// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command fetch fills a cache with the MPEG audio conformance streams of ISO/IEC 14496-26 and
// their references, which the conformance test reads from the directory MP3_DECODER_CONFORMANCE
// names.
//
//	go run ./internal/conformance/fetch -dir DIR
//
// A complete cache needs no network.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
)

// Exit codes.
const (
	// exitFailure reports a failed fetch.
	exitFailure = 1
	// exitUsage reports bad flags.
	exitUsage = 2
)

// requestTimeout bounds each request, which carries at most 1 MiB.
const requestTimeout = 2 * time.Minute

// errNoDir is returned when -dir is missing or arguments follow the flags.
var errNoDir = errors.New("fetch: -dir names the cache, and no arguments follow the flags")

// main fetches, then exits with the outcome.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)

	stop()
	os.Exit(code)
}

// run parses the flags and fills the cache.
//
// Parameters:
//   - ctx: cancels the fetch.
//   - args: the arguments after the command's name.
//   - stdout: where the outcome goes.
//   - stderr: where errors and usage go.
//
// Returns:
//   - int: 0 on success, 1 when the fetch fails, 2 for bad flags.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("fetch", flag.ContinueOnError)
	flags.SetOutput(stderr)

	dir := flags.String("dir", "", "the cache to fill")
	base := flags.String("url", conformance.ArchiveURL, "where the conformance archives are published")

	err := flags.Parse(args)
	if err != nil {
		return exitUsage
	}

	if *dir == "" || flags.NArg() > 0 {
		_, _ = fmt.Fprintln(stderr, errNoDir)

		return exitUsage
	}

	client := new(http.Client)

	client.Timeout = requestTimeout

	fetched, err := conformance.Fetch(ctx, client, *base, *dir, conformance.Streams())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "fetch:", err)

		return exitFailure
	}

	_, _ = fmt.Fprintf(stdout, "conformance streams ready in %s, %d fetched\n", *dir, fetched)

	return 0
}
