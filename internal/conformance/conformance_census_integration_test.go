// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/layer3"
)

// features lists what the census looks for, in the order it shows them.
var features = []string{
	"Layer I", "Layer II", "Layer III", "MPEG-1", "MPEG-2", "MPEG-2.5",
	"8 kHz", "11.025 kHz", "12 kHz", "16 kHz", "22.05 kHz", "24 kHz", "32 kHz", "44.1 kHz", "48 kHz",
	"bitrate changes", "free format", "CRC",
	"mono", "stereo", "dual channel", "joint stereo, neither", "joint stereo, mid/side",
	"joint stereo, intensity", "joint stereo, mid/side and intensity", "joint stereo from subband 4",
	"joint stereo from subband 8", "joint stereo from subband 12", "joint stereo from subband 16",
	"a change between mono and stereo",
	"normal blocks", "start blocks", "short blocks", "stop blocks", "mixed short blocks",
	"mixed start or stop blocks", "scfsi", "preflag bit of MPEG-1", "scalefac_scale", "big_values tables with linbits",
	"count1 table B", "main data before the stream starts", "a last frame cut short",
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
		found[[]string{"", "Layer I", "Layer II", "Layer III"}[head.Layer]] = true
		found[strconv.FormatFloat(float64(head.SampleRate)/1000, 'f', -1, 64)+" kHz"] = true
		found[modeName(head)] = true
		found["bitrate changes"] = found["bitrate changes"] || head.Bitrate != first.Bitrate
		found["CRC"] = found["CRC"] || head.Protected
		found["a change between mono and stereo"] = found["a change between mono and stereo"] ||
			channels != 0 && channels != head.Channels()

		channels = head.Channels()

		if head.Layer != frame.Layer3 || len(own) < head.DataAt() || side.Parse(head, own) != nil {
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
	switch {
	case head.Mode == frame.Mono:
		return "mono"
	case head.Mode == frame.Stereo:
		return "stereo"
	case head.Mode == frame.DualChannel:
		return "dual channel"
	case head.Layer != frame.Layer3:
		// The mode extension of Layers I and II names the subband the intensity stereo starts at.
		return "joint stereo from subband " + strconv.Itoa(4*(int(head.Extension)+1))
	default:
		return []string{
			"joint stereo, neither", "joint stereo, intensity", "joint stereo, mid/side",
			"joint stereo, mid/side and intensity",
		}[head.Extension]
	}
}
