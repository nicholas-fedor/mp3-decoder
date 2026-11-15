// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package layer12

import (
	"errors"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// Decoder decodes the audio of a stream of Layer I frames.
//
// It keeps the state of each channel's synthesis filterbank, which carries each frame's samples
// into the next.
type Decoder struct {
	// stream reads the frame's audio data.
	stream bits.Reader
	// synth holds each channel's polyphase synthesis filterbank.
	synth synth.Filterbanks
	// steps holds, for each channel and subband, the size of one quantization step scaled by the
	// subband's scalefactor, or 0 for a subband without samples.
	steps [2][synth.Subbands]float64
	// samples holds one set of subband samples for each channel.
	samples [2][synth.Subbands]float64
	// widths holds the number of bits of each sample, for each channel and subband, or 0 for a
	// subband without samples.
	widths [2][synth.Subbands]int
}

// Frame layout.
const (
	// bitsPerByte is the number of bits in a byte.
	bitsPerByte = 8
	// boundStep is the number of subbands each step of the mode extension moves the bound by.
	boundStep = 4
)

// ErrAllocation is returned for a frame that gives a subband the forbidden bit allocation.
var ErrAllocation = errors.New("layer12: forbidden bit allocation")

// Decode decodes the audio of one Layer I frame.
//
// A frame whose CRC does not match the bits it covers is decoded all the same. A frame with
// another number of channels than the frame before it starts both channels' filterbanks afresh,
// as a new stream would. A mono frame gives each sample to both channels. The samples are not
// clipped.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs, at least as many as the frame holds.
//   - head: the frame's header.
//   - data: the frame's bytes from its header to its end.
//
// Returns:
//   - bool: whether the frame's CRC matches its bit allocation, or the frame has no CRC.
//   - error: [ErrAllocation], and then dst is left as it was.
func (d *Decoder) Decode(dst [][2]float64, head frame.Header, data []byte) (bool, error) {
	start := bitsPerByte * head.DataAt()
	channels, bound := head.Channels(), jointBound(head)

	d.stream.Reset(data, start)

	allowed := d.readAllocation(channels, bound)
	intact := head.CheckCRC(data, d.stream.Pos()-start)

	if !allowed {
		return intact, ErrAllocation
	}

	d.synth.Follow(channels)
	d.readScalefactors(channels)

	for set := range layer1Sets {
		d.readSet(channels, bound)

		pcm := dst[set*synth.Subbands : (set+1)*synth.Subbands]

		for ch := range channels {
			var out [synth.Subbands]float64

			d.synth.Channel[ch].Synthesize(&out, &d.samples[ch])

			for sample, value := range out {
				pcm[sample][ch] = value
			}
		}
	}

	if channels == 1 {
		for at := range layer1Sets * synth.Subbands {
			dst[at][1] = dst[at][0]
		}
	}

	return intact, nil
}

// Reset readies the decoder for a stream from its start, with fresh filterbanks.
func (d *Decoder) Reset() {
	var fresh Decoder

	*d = fresh
}

// jointBound returns the first subband whose samples both channels share.
//
// In joint stereo the mode extension sets it to 4, 8, 12 or 16. In every other mode each channel
// keeps its own samples in all 32 subbands.
//
// Parameters:
//   - head: the frame's header.
//
// Returns:
//   - int: the subband.
func jointBound(head frame.Header) int {
	if head.Mode != frame.JointStereo {
		return synth.Subbands
	}

	return boundStep * (int(head.Extension) + 1)
}
