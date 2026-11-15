// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
	"errors"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/hybrid"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// Decoder decodes the audio of a stream of Layer III frames.
//
// It keeps what a frame passes to the frames after it: the bit reservoir, the scalefactors that
// scfsi reuses, and the state of each channel's filterbanks.
type Decoder struct {
	// stream reads the main data.
	stream bits.Reader
	// spectrum holds the quantized values of the granule being decoded.
	spectrum Spectrum
	// synth holds each channel's polyphase synthesis filterbank.
	synth [2]synth.Filterbank
	// lines holds a granule's spectral lines for each channel.
	lines [2][granuleLines]float64
	// hybrid holds each channel's hybrid filterbank.
	hybrid [2]hybrid.Filterbank
	// subbands holds a granule's subband samples for one channel, at [sample][subband].
	subbands [hybrid.Samples][hybrid.Subbands]float64
	// reservoir holds the main data that later frames reach back to.
	reservoir Reservoir
	// side holds the side information of the frame being decoded.
	side SideInfo
	// factors holds each channel's scalefactors.
	factors [2]Scalefactors
}

// granulePCM is the number of PCM samples a granule gives for each channel.
const granulePCM = hybrid.Samples * synth.Subbands

// ErrVersion is returned for an MPEG-2 or MPEG-2.5 frame, whose scalefactors the decoder does not
// read.
var ErrVersion = errors.New("layer3: only MPEG-1 frames are decoded")

// Decode decodes the audio of one frame.
//
// The frame's main data may begin in the main data of the frames before it. A granule whose main
// data begins before the bytes the decoder holds, as after joining a stream midway, gives silence,
// but the filterbanks still run, so the granules after it carry on smoothly. A mono frame gives
// each sample to both channels. The samples are not clipped.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs, at least as many as the frame holds.
//   - head: the frame's header.
//   - data: the frame's bytes from its header to its end.
//
// Returns:
//   - error: [ErrVersion], [ErrShortFrame] or [ErrBlockType], and then dst is left as it was.
func (d *Decoder) Decode(dst [][2]float64, head frame.Header, data []byte) error {
	if head.Version != frame.MPEG1 {
		return ErrVersion
	}

	err := d.side.Parse(head, data)
	if err != nil {
		return err
	}

	main, missing := d.reservoir.MainData(d.side.MainDataBegin, data[head.DataAt():])
	start := -bitsPerByte * missing

	for gr := range mpeg1Granules {
		for ch := range head.Channels() {
			d.requantize(main, head, gr, ch, start)

			start += d.side.Granules[gr][ch].Part23Length
		}

		if head.Mode != frame.Mono {
			Stereo(head, &d.lines[0], &d.lines[1], &d.side.Granules[gr][1], &d.factors[1])
		}

		for ch := range head.Channels() {
			d.synthesize(dst[gr*granulePCM:(gr+1)*granulePCM], head, gr, ch)
		}
	}

	if head.Mode == frame.Mono {
		for at := range mpeg1Granules * granulePCM {
			dst[at][1] = dst[at][0]
		}
	}

	return nil
}

// requantize decodes one channel's granule into its spectral lines, or silence when its main data
// begins before the bytes the decoder holds.
//
// Parameters:
//   - main: the frame's main data, from the first byte the decoder holds.
//   - head: the frame's header.
//   - gr: the granule.
//   - ch: the channel.
//   - start: the position of the granule's first bit in main, below 0 when it is missing.
func (d *Decoder) requantize(main []byte, head frame.Header, gr, ch, start int) {
	if start < 0 {
		d.lines[ch] = [granuleLines]float64{}

		return
	}

	info := &d.side.Granules[gr][ch]

	d.stream.Reset(main, start)
	d.factors[ch].ReadMPEG1(&d.stream, &d.side, gr, ch)
	d.spectrum.Read(&d.stream, head, info, start+info.Part23Length)
	d.spectrum.Requantize(&d.lines[ch], info, &d.factors[ch])
}

// synthesize turns one channel's spectral lines into the granule's PCM samples.
//
// Parameters:
//   - dst: the granule's PCM samples.
//   - head: the frame's header.
//   - gr: the granule.
//   - ch: the channel.
func (d *Decoder) synthesize(dst [][2]float64, head frame.Header, gr, ch int) {
	info := &d.side.Granules[gr][ch]

	Reorder(head, &d.lines[ch], info)
	d.hybrid[ch].Transform(&d.subbands, &d.lines[ch], hybridBlock(info, bandsAt(head.SampleRate)))

	var pcm [synth.Subbands]float64

	for at := range d.subbands {
		d.synth[ch].Synthesize(&pcm, &d.subbands[at])

		for sample, value := range pcm {
			dst[at*synth.Subbands+sample][ch] = value
		}
	}
}

// hybridBlock returns which subbands of a granule hold long blocks, and their window.
//
// Mixed blocks hold long blocks up to where short band 3 starts, 2 subbands, or 4 at 8 kHz, and
// they take the normal window.
//
// Parameters:
//   - info: the granule's side information.
//   - partition: the bands of the frame's sampling frequency.
//
// Returns:
//   - hybrid.Block: the block.
func hybridBlock(info *GranuleInfo, partition *bands) hybrid.Block {
	switch {
	case info.BlockType == StartBlock:
		return hybrid.Block{Long: hybrid.Subbands, Window: hybrid.Start}
	case info.BlockType == StopBlock:
		return hybrid.Block{Long: hybrid.Subbands, Window: hybrid.Stop}
	case info.BlockType != ShortBlocks:
		return hybrid.Block{Long: hybrid.Subbands, Window: hybrid.Normal}
	case info.MixedBlock:
		return hybrid.Block{Long: partition.mixedEnd() / hybrid.Samples, Window: hybrid.Normal}
	default:
		return hybrid.Block{Long: 0, Window: hybrid.Normal}
	}
}
