// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer3

import (
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
	// channels is the number of channels of the frame decoded last, or 0 before the first.
	channels int
}

// granulePCM is the number of PCM samples a granule gives for each channel.
const granulePCM = hybrid.Samples * synth.Subbands

// Decode decodes the audio of one frame.
//
// The frame's main data may begin in the main data of the frames before it. A frame whose main
// data begins before the bytes the decoder holds, as after joining a stream midway, gives silence
// in both granules, but its main data still enters the bit reservoir and the filterbanks still
// run, so the frames after it carry on smoothly. A frame with another number of channels than the
// frame before it starts both channels' filterbanks afresh, as a new stream would, while the bit
// reservoir carries on. A mono frame gives each sample to both channels. The samples are not
// clipped.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs, at least as many as the frame holds.
//   - head: the frame's header.
//   - data: the frame's bytes from its header to its end.
//
// Returns:
//   - error: [ErrShortFrame] or [ErrBlockType], and then dst is left as it was.
func (d *Decoder) Decode(dst [][2]float64, head frame.Header, data []byte) error {
	err := d.side.Parse(head, data)
	if err != nil {
		return err
	}

	d.followChannels(head)

	main, missing := d.reservoir.MainData(d.side.MainDataBegin, data[head.DataAt():])
	start := 0

	granules := head.Samples() / granulePCM

	for gr := range granules {
		start = d.granule(dst[gr*granulePCM:(gr+1)*granulePCM], main, head, gr, start, missing)
	}

	if head.Mode == frame.Mono {
		for at := range granules * granulePCM {
			dst[at][1] = dst[at][0]
		}
	}

	return nil
}

// followChannels starts both channels' filterbanks afresh when a frame has another number of
// channels than the frame before it, then records the frame's.
//
// Parameters:
//   - head: the frame's header.
func (d *Decoder) followChannels(head frame.Header) {
	if d.channels != 0 && d.channels != head.Channels() {
		for ch := range d.hybrid {
			d.hybrid[ch].Reset()
			d.synth[ch].Reset()
		}
	}

	d.channels = head.Channels()
}

// granule decodes one granule of every channel into its PCM samples.
//
// Parameters:
//   - dst: the granule's PCM samples.
//   - main: the frame's main data.
//   - head: the frame's header.
//   - gr: the granule.
//   - start: the position of the granule's first bit in main.
//   - missing: how many bytes of the frame's main data the decoder lacks, which silence the
//     granule when above 0.
//
// Returns:
//   - int: the position of the next granule's first bit in main.
func (d *Decoder) granule(dst [][2]float64, main []byte, head frame.Header, gr, start, missing int) int {
	for ch := range head.Channels() {
		if missing > 0 {
			d.lines[ch] = [granuleLines]float64{}
		} else {
			d.requantize(main, head, gr, ch, start)
		}

		start += d.side.Granules[gr][ch].Part23Length
	}

	if head.Mode != frame.Mono {
		Stereo(head, &d.lines[0], &d.lines[1], &d.side.Granules[gr][1], &d.factors[1])
	}

	for ch := range head.Channels() {
		d.synthesize(dst, head, gr, ch)
	}

	return start
}

// requantize decodes one channel's granule into its spectral lines.
//
// Parameters:
//   - main: the frame's main data.
//   - head: the frame's header.
//   - gr: the granule.
//   - ch: the channel.
//   - start: the position of the granule's first bit in main.
func (d *Decoder) requantize(main []byte, head frame.Header, gr, ch, start int) {
	info := &d.side.Granules[gr][ch]

	d.stream.Reset(main, start)
	d.factors[ch].Read(&d.stream, head, &d.side, gr, ch)
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

// hybridBlock returns which subbands of a granule hold long blocks, and their windows.
//
// The long part of a mixed block spans the subbands below where short band 3 starts, 2 subbands,
// or 4 at 8 kHz, and takes the normal window. A start or stop block with the mixed flag set gives
// those subbands the normal window too.
//
// Parameters:
//   - info: the granule's side information.
//   - partition: the bands of the frame's sampling frequency.
//
// Returns:
//   - hybrid.Block: the block.
func hybridBlock(info *GranuleInfo, partition *bands) hybrid.Block {
	mixed := 0
	if info.MixedBlock {
		mixed = partition.mixedEnd() / hybrid.Samples
	}

	switch info.BlockType {
	case StartBlock:
		return hybrid.Block{Long: hybrid.Subbands, Mixed: mixed, Window: hybrid.Start}
	case StopBlock:
		return hybrid.Block{Long: hybrid.Subbands, Mixed: mixed, Window: hybrid.Stop}
	case ShortBlocks:
		return hybrid.Block{Long: mixed, Mixed: mixed, Window: hybrid.Normal}
	default:
		return hybrid.Block{Long: hybrid.Subbands, Mixed: 0, Window: hybrid.Normal}
	}
}
