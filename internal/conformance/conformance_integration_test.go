// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance_test

import (
	"bytes"
	"crypto/sha256"
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
	// features holds the features the census found in the stream.
	features map[string]bool
	// stream is the stream.
	stream conformance.Stream
	// differences holds the differences of each reference's channels, or nil for a reference
	// that could not be measured.
	differences [][]conformance.Difference
	// refusals holds why each reference could not be measured, or nil.
	refusals []error
	// failed counts the frames that failed to decode.
	failed int
	// digest is the hash of the decoded samples.
	digest [sha256.Size]byte
}

// region is a part of the document that the test writes, between two markers.
type region struct {
	// start marks the region's start.
	start string
	// end marks the region's end.
	end string
	// text is what the region holds.
	text string
}

// resultsPath is the document that records the results and the census.
const resultsPath = "../../docs/conformance.md"

// update rewrites the results and the census in the document instead of checking them.
var update = flag.Bool("update", false, "rewrite the results and the census in docs/conformance.md")

// TestConformance decodes every conformance stream in the cache that MP3_DECODER_CONFORMANCE names,
// measures it against its references, takes a census of the features it uses, and checks both
// against what docs/conformance.md records.
//
// With -update it records them there instead.
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

// record checks the results and the census against those docs/conformance.md records, or with
// -update records them there.
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

	text := string(document)

	for _, part := range []region{
		{start: "<!-- results start -->\n", end: "<!-- results end -->\n", text: render(outcomes)},
		{start: "<!-- census start -->\n", end: "<!-- census end -->\n", text: census(outcomes)},
	} {
		before, rest, found := strings.Cut(text, part.start)
		recorded, after, ended := strings.Cut(rest, part.end)

		if !assert.True(t, found && ended, "the markers %q are missing", part.start) {
			return
		}

		if !*update {
			assert.Equal(t, recorded, part.text, "%s changed: rerun the test with -update", part.start)
		}

		text = before + part.start + part.text + part.end + after
	}

	if *update {
		err = os.WriteFile(resultsPath, []byte(text), 0o600)
		if err != nil {
			t.Errorf("record the results: %v", err)
		}
	}
}

// measure decodes a stream from the cache and measures it against each of its references.
//
// Every frame of the stream must match its CRC.
func measure(t *testing.T, dir string, stream conformance.Stream) outcome {
	t.Helper()

	data, err := stream.Read(dir)
	require.NoError(t, err)

	found := outcome{
		opening: nil, failure: nil, features: survey(data), stream: stream, differences: make([][]conformance.Difference, len(stream.References)),
		refusals: make([]error, len(stream.References)), failed: 0, digest: [sha256.Size]byte{},
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

	if decoder != nil {
		assert.Zero(t, decoder.Stats().CRCMismatches, "%s: frames whose CRC does not match", stream.Name)
	}

	found.digest = conformance.Digest(decoded)

	for at, ref := range stream.References {
		file, err := os.ReadFile(ref.Path(dir))
		require.NoError(t, err)

		wave, err := conformance.ParseWAV(file)
		require.NoError(t, err)

		found.differences[at], found.refusals[at] = ref.Measure(decoded, wave)
	}

	return found
}

// render returns the results as Markdown: how many streams conform, one hash of every stream's
// decoded samples, then a row for each channel of each reference.
func render(outcomes []outcome) string {
	var rows strings.Builder

	conforming := 0
	digests := sha256.New()

	for _, found := range outcomes {
		if conforms(found) {
			conforming++
		}

		digests.Write(found.digest[:])

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
					result += ", " + failures(found)
				}

				fmt.Fprintf(&rows, "| %s | %s | %s | %.1e | %.1e | %s |\n", found.stream.Name, name,
					[]string{"left", "right"}[ref.Channels[channel]], difference.RMS, difference.Max, result)
			}
		}
	}

	return fmt.Sprintf("%d of %d streams decode at full accuracy.\n\n", conforming, len(outcomes)) +
		fmt.Sprintf("The decoded samples of all %d streams hash to `%x`.\n\n", len(outcomes), digests.Sum(nil)) +
		"| Stream | Reference | Channel | RMS | Largest | Result |\n| --- | --- | --- | --- | --- | --- |\n" +
		rows.String()
}

// refusal words why a reference could not be measured: the error from opening the stream, the
// first error from decoding a frame when frames failed, or else the measurement's own error.
func refusal(found outcome, at int) string {
	switch {
	case found.opening != nil:
		return fmt.Sprintf("not opened, with the error %q", found.opening)
	case found.failed > 0:
		return "not decoded, " + failures(found)
	default:
		return fmt.Sprintf("not measured, with the error %q", found.refusals[at])
	}
}

// failures words how many frames failed to decode, and the first one's error.
func failures(found outcome) string {
	frames := "frames"
	if found.failed == 1 {
		frames = "frame"
	}

	return fmt.Sprintf("%d %s failed, the first with the error %q", found.failed, frames, found.failure)
}

// conforms reports whether a stream decodes at full accuracy, with every reference, or one of
// its alternatives, matching on every channel.
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
