// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package synth

import "testing"

// FuzzSynthesize synthesizes runs of subband samples drawn from any bytes and checks every PCM
// sample against the oracle that follows the published flow.
//
// Each byte gives a subband sample between -1 and 1, 32 bytes to a call, and the last call takes
// zeros past the bytes. The seed corpus holds a ramp, alternating extremes over 20 calls, and
// silence.
func FuzzSynthesize(f *testing.F) {
	ramp := make([]byte, Subbands)
	for at := range ramp {
		ramp[at] = byte(8 * at)
	}

	extremes := make([]byte, 20*Subbands)
	for at := range extremes {
		extremes[at] = byte(0x7f + at%2*2)
	}

	f.Add(ramp)
	f.Add(extremes)
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			bank   Filterbank
			oracle [historySize]float64
		)

		for call := 0; call*Subbands < max(len(data), 1) && call < 40; call++ {
			var samples [Subbands]float64

			for k := range samples {
				if at := call*Subbands + k; at < len(data) {
					samples[k] = float64(int8(data[at])) / 128
				}
			}

			want := synthesizeOracle(&oracle, &samples)

			var got [Subbands]float64

			bank.Synthesize(&got, &samples)
			assertPCM(t, &want, &got, call)
		}
	})
}
