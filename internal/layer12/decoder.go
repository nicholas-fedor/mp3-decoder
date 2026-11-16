// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package layer12

import (
	"errors"

	"github.com/nicholas-fedor/mp3-decoder/frame"
	"github.com/nicholas-fedor/mp3-decoder/internal/bits"
	"github.com/nicholas-fedor/mp3-decoder/internal/synth"
)

// Decoder decodes the audio of a stream of Layer I or Layer II frames.
//
// It keeps the state of each channel's synthesis filterbank, which carries each frame's samples
// into the next.
type Decoder struct {
	// stream reads the frame's audio data.
	stream bits.Reader
	// synth holds each channel's polyphase synthesis filterbank.
	synth synth.Filterbanks
	// steps holds, for each channel and subband, the size of one quantization step scaled by the
	// subband's scalefactor.
	steps [2][synth.Subbands]float64
	// samples holds up to three sets of subband samples for each channel.
	samples [granuleSets][2][synth.Subbands]float64
	// classes holds the quantization class of each channel's subbands, with no steps for a subband
	// without samples.
	classes [2][synth.Subbands]quantClass
	// factors holds the scalefactor indices of each channel's subbands in the three parts of a
	// Layer II frame.
	factors [2][synth.Subbands][layer2Parts]uint32
	// selections holds how many scalefactors each channel's subbands carry in a Layer II frame.
	selections [2][synth.Subbands]uint32
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

// Decode decodes the audio of one Layer I or Layer II frame.
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
//   - bool: whether the frame's CRC matches the bits it covers, or the frame has no CRC.
//   - error: [ErrAllocation] for a Layer I frame, and then dst is left as it was.
func (d *Decoder) Decode(dst [][2]float64, head frame.Header, data []byte) (bool, error) {
	d.stream.Reset(data, bitsPerByte*head.DataAt())

	var (
		intact bool
		err    error
	)

	if head.Layer == frame.Layer2 {
		intact = d.decodeLayer2(dst, head, data)
	} else {
		intact, err = d.decodeLayer1(dst, head, data)
	}

	if err == nil && head.Mode == frame.Mono {
		for at := range head.Samples() {
			dst[at][1] = dst[at][0]
		}
	}

	return intact, err
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

// decodeLayer1 decodes the audio of one Layer I frame, one set of subband samples at a time.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs.
//   - head: the frame's header.
//   - data: the frame's bytes from its header to its end.
//
// Returns:
//   - bool: whether the frame's CRC matches its bit allocation, or the frame has no CRC.
//   - error: [ErrAllocation], and then dst is left as it was.
func (d *Decoder) decodeLayer1(dst [][2]float64, head frame.Header, data []byte) (bool, error) {
	start, channels, bound := d.stream.Pos(), head.Channels(), jointBound(head)

	allowed := d.readAllocation(channels, bound)
	intact := head.CheckCRC(data, d.stream.Pos()-start)

	if !allowed {
		return intact, ErrAllocation
	}

	d.synth.Follow(channels)
	d.readScalefactors(channels)

	for set := range layer1Sets {
		d.readSet(channels, bound)
		d.synthesize(dst[set*synth.Subbands:], 1, channels)
	}

	return intact, nil
}

// synthesize turns the sets of subband samples held into PCM samples.
//
// Parameters:
//   - dst: the PCM samples as left and right pairs, from those of the first set.
//   - sets: the number of sets.
//   - channels: the frame's number of channels.
func (d *Decoder) synthesize(dst [][2]float64, sets, channels int) {
	for set := range sets {
		pcm := dst[set*synth.Subbands : (set+1)*synth.Subbands]

		for ch := range channels {
			var out [synth.Subbands]float64

			d.synth.Channel[ch].Synthesize(&out, &d.samples[set][ch])

			for sample, value := range out {
				pcm[sample][ch] = value
			}
		}
	}
}
