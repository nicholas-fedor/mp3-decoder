// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
)

// outcome is what decoding and measuring one stream found.
type outcome struct {
	// opening is the error from opening the stream, or nil.
	opening error
	// failure is the first error from decoding a frame, or nil.
	failure error
	// stream is the stream.
	stream conformance.Stream
	// differences holds the differences of each reference's channels, or nil for a reference
	// that could not be measured.
	differences [][]conformance.Difference
	// refusals holds why each reference could not be measured, or nil.
	refusals []error
	// failed counts the frames that failed to decode.
	failed int
}

// Where the results go.
const (
	// resultsPath is the document that records the results.
	resultsPath = "../../docs/conformance.md"
	// resultsStart marks the start of the results in the document.
	resultsStart = "<!-- results start -->\n"
	// resultsEnd marks the end of the results in the document.
	resultsEnd = "<!-- results end -->\n"
)

// update rewrites the results in the document instead of checking them.
var update = flag.Bool("update", false, "rewrite the results in docs/conformance.md")

// TestConformance decodes every conformance stream in the cache that MP3_DECODER_CONFORMANCE names,
// measures it against its references, and checks the results against those docs/conformance.md
// records. With -update it records them there instead.
//
// A file missing from the cache fails the test.
func TestConformance(t *testing.T) {
	t.Parallel()

	dir := os.Getenv("MP3_DECODER_CONFORMANCE")
	if dir == "" {
		t.Skip("MP3_DECODER_CONFORMANCE names no cache of conformance streams")
	}

	streams := conformance.Streams()
	outcomes := make([]outcome, len(streams))

	// Cleanups run once every subtest has finished.
	t.Cleanup(func() {
		if !t.Failed() {
			record(t, outcomes)
		}
	})

	for at, stream := range streams {
		t.Run(stream.Name, func(t *testing.T) {
			t.Parallel()

			outcomes[at] = measure(t, dir, stream)
		})
	}
}

// record checks the results against those docs/conformance.md records, or with -update records
// them there.
//
// It runs in a cleanup, where a failure that stops the test would lose the test's report, so it
// reports failures and returns.
func record(t *testing.T, outcomes []outcome) {
	t.Helper()

	document, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Errorf("read the results: %v", err)

		return
	}

	before, rest, found := strings.Cut(string(document), resultsStart)
	recorded, after, ended := strings.Cut(rest, resultsEnd)

	if !assert.True(t, found && ended, "the results markers are missing") {
		return
	}

	results := render(outcomes)
	if *update {
		err = os.WriteFile(resultsPath, []byte(before+resultsStart+results+resultsEnd+after), 0o600)
		if err != nil {
			t.Errorf("record the results: %v", err)
		}

		return
	}

	assert.Equal(t, recorded, results, "the results changed: rerun the test with -update")
}

// measure decodes a stream from the cache and measures it against each of its references.
func measure(t *testing.T, dir string, stream conformance.Stream) outcome {
	t.Helper()

	data, err := os.ReadFile(stream.Path(dir))
	require.NoError(t, err)

	found := outcome{
		opening: nil, failure: nil, stream: stream, differences: make([][]conformance.Difference, len(stream.References)),
		refusals: make([]error, len(stream.References)), failed: 0,
	}

	var decoded [][2]float64

	decoder, err := mp3.OpenStream(t.Context(), bytes.NewReader(data))
	if err != nil {
		found.opening = err
	}

	for buffer := make([][2]float64, 4096); decoder != nil; {
		count, err := decoder.Read(buffer)
		decoded = append(decoded, buffer[:count]...)

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil && found.failure == nil {
			found.failure = err
		}

		if err != nil {
			found.failed++
		}
	}

	for at, ref := range stream.References {
		file, err := os.ReadFile(ref.Path(dir))
		require.NoError(t, err)

		wave, err := conformance.ParseWAV(file)
		require.NoError(t, err)

		found.differences[at], found.refusals[at] = ref.Measure(decoded, wave)
	}

	return found
}

// render returns the results as Markdown: how many streams conform, then a row for each channel of
// each reference.
func render(outcomes []outcome) string {
	var rows strings.Builder

	conforming := 0

	for _, found := range outcomes {
		if conforms(found) {
			conforming++
		}

		for at, ref := range found.stream.References {
			name := ref.Name
			if ref.Alternative {
				name += " (alternative)"
			}

			if found.differences[at] == nil {
				fmt.Fprintf(&rows, "| %s | %s | | | | %s |\n", found.stream.Name, name, refusal(found, at))

				continue
			}

			for channel, difference := range found.differences[at] {
				result := "full accuracy"
				if !difference.Full() {
					result = "outside full accuracy"
				}

				if found.failed > 0 {
					result += fmt.Sprintf(", %d frames failed: %v", found.failed, found.failure)
				}

				fmt.Fprintf(&rows, "| %s | %s | %s | %.1e | %.1e | %s |\n", found.stream.Name, name,
					[]string{"left", "right"}[ref.Channels[channel]], difference.RMS, difference.Max, result)
			}
		}
	}

	return fmt.Sprintf("%d of %d streams decode at full accuracy.\n\n", conforming, len(outcomes)) +
		"| Stream | Reference | Channel | RMS | Largest | Result |\n| --- | --- | --- | --- | --- | --- |\n" +
		rows.String()
}

// refusal words why a reference could not be measured: the error from opening the stream, the
// first error from decoding a frame when frames failed, or else the measurement's own error.
func refusal(found outcome, at int) string {
	switch {
	case found.opening != nil:
		return fmt.Sprintf("not opened: %v", found.opening)
	case found.failed > 0:
		return fmt.Sprintf("not decoded, %d frames failed: %v", found.failed, found.failure)
	default:
		return fmt.Sprintf("not measured: %v", found.refusals[at])
	}
}

// conforms reports whether a stream decodes at full accuracy: every reference, or one of its
// alternatives, matches on every channel.
func conforms(found outcome) bool {
	matched := false

	for at, ref := range found.stream.References {
		full := found.differences[at] != nil

		for _, difference := range found.differences[at] {
			full = full && difference.Full()
		}

		switch {
		case !ref.Alternative && at > 0 && !matched:
			return false
		case !ref.Alternative:
			matched = full
		default:
			matched = matched || full
		}
	}

	return matched
}
