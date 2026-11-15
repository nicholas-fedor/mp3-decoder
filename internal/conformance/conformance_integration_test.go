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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder"
	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
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

// features lists what the census looks for, in the order it shows them.
var features = []string{
	"MPEG-1", "MPEG-2", "MPEG-2.5",
	"8 kHz", "11.025 kHz", "12 kHz", "16 kHz", "22.05 kHz", "24 kHz", "32 kHz", "44.1 kHz", "48 kHz",
	"bitrate changes", "free format", "CRC",
	"mono", "stereo", "dual channel", "joint stereo, neither", "joint stereo, mid/side",
	"joint stereo, intensity", "joint stereo, mid/side and intensity", "a change between mono and stereo",
	"normal blocks", "start blocks", "short blocks", "stop blocks", "mixed short blocks",
	"mixed start or stop blocks", "scfsi", "preflag bit of MPEG-1", "scalefac_scale", "big_values tables with linbits",
	"count1 table B", "main data before the stream starts", "a last frame cut short",
}

// update rewrites the results and the census in the document instead of checking them.
var update = flag.Bool("update", false, "rewrite the results and the census in docs/conformance.md")

// TestConformance decodes every conformance stream in the cache that MP3_DECODER_CONFORMANCE names,
// measures it against its references, takes a census of the features it uses, and checks both
// against what docs/conformance.md records. With -update it records them there instead.
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
func measure(t *testing.T, dir string, stream conformance.Stream) outcome {
	t.Helper()

	data, err := os.ReadFile(stream.Path(dir))
	require.NoError(t, err)

	found := outcome{
		opening: nil, failure: nil, features: survey(data), stream: stream, differences: make([][]conformance.Difference, len(stream.References)),
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

// census returns, for each feature, the streams that use it, as a Markdown table.
func census(outcomes []outcome) string {
	var rows strings.Builder

	rows.WriteString("| Feature | Streams |\n| --- | --- |\n")

	for _, feature := range features {
		var names []string

		for _, found := range outcomes {
			if found.features[feature] {
				names = append(names, found.stream.Name)
			}
		}

		listed := strings.Join(names, ", ")

		switch len(names) {
		case 0:
			listed = "none"
		case len(outcomes):
			listed = "all " + strconv.Itoa(len(outcomes))
		default:
		}

		fmt.Fprintf(&rows, "| %s | %s |\n", feature, listed)
	}

	return rows.String()
}

// survey walks a stream's frames and returns the features its headers and side information use.
func survey(data []byte) map[string]bool {
	found := map[string]bool{}

	at, first, ok := frame.Find(data, nil)

	var (
		reservoir layer3.Reservoir
		side      layer3.SideInfo
	)

	for channels := 0; ok && at+frame.HeaderSize <= len(data); {
		head, parsed := frame.Parse(data[at:])
		if !parsed || !head.SameStream(first) {
			break
		}

		if head.Free {
			head.Bitrate = first.Bitrate
			found["free format"] = true
		}

		own := data[at:min(at+head.Size(), len(data))]
		at += head.Size()

		found["a last frame cut short"] = found["a last frame cut short"] || at > len(data)
		found[[]string{"", "MPEG-1", "MPEG-2", "MPEG-2.5"}[head.Version]] = true
		found[strconv.FormatFloat(float64(head.SampleRate)/1000, 'f', -1, 64)+" kHz"] = true
		found[modeName(head)] = true
		found["bitrate changes"] = found["bitrate changes"] || head.Bitrate != first.Bitrate
		found["CRC"] = found["CRC"] || head.Protected
		found["a change between mono and stereo"] = found["a change between mono and stereo"] ||
			channels != 0 && channels != head.Channels()

		channels = head.Channels()

		if len(own) < head.DataAt() || side.Parse(head, own) != nil {
			continue
		}

		_, missing := reservoir.MainData(side.MainDataBegin, own[head.DataAt():])
		found["main data before the stream starts"] = found["main data before the stream starts"] || missing > 0

		surveyGranules(found, head, &side)
	}

	return found
}

// surveyGranules adds the features of a frame's granules.
func surveyGranules(found map[string]bool, head frame.Header, side *layer3.SideInfo) {
	for gr := range head.Samples() / 576 {
		for ch := range head.Channels() {
			info := side.Granules[gr][ch]
			kind := []string{"normal blocks", "start blocks", "short blocks", "stop blocks"}[info.BlockType]

			switch {
			case info.MixedBlock && info.BlockType == layer3.ShortBlocks:
				kind = "mixed short blocks"
			case info.MixedBlock:
				kind = "mixed start or stop blocks"
			default:
			}

			found[kind] = true
			found["preflag bit of MPEG-1"] = found["preflag bit of MPEG-1"] || info.Preflag
			found["scalefac_scale"] = found["scalefac_scale"] || info.ScalefacScale
			found["count1 table B"] = found["count1 table B"] || info.Count1TableSelect == 1
			found["scfsi"] = found["scfsi"] || side.SCFSI[ch] != [4]bool{}

			for _, table := range info.TableSelect {
				found["big_values tables with linbits"] = found["big_values tables with linbits"] || table >= 16
			}
		}
	}
}

// modeName names a frame's channel mode, and for joint stereo the tools it uses.
func modeName(head frame.Header) string {
	switch head.Mode {
	case frame.Mono:
		return "mono"
	case frame.Stereo:
		return "stereo"
	case frame.DualChannel:
		return "dual channel"
	default:
		return []string{
			"joint stereo, neither", "joint stereo, intensity", "joint stereo, mid/side",
			"joint stereo, mid/side and intensity",
		}[head.Extension]
	}
}
