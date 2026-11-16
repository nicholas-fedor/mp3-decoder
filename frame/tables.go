// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame

// kilo turns kbit/s into bit/s.
const kilo = 1000

// versions maps the two version bits to a version, or 0 for the reserved value.
var versions = [4]Version{MPEG25, 0, MPEG2, MPEG1}

// layers maps the two layer bits to a layer, or 0 for the reserved value.
var layers = [4]Layer{0, Layer3, Layer2, Layer1}

// modes maps the two mode bits to a mode.
var modes = [4]Mode{Stereo, JointStereo, DualChannel, Mono}

// sampleRates maps a version and a sampling frequency index to a rate in Hz.
var sampleRates = [4][3]int{
	MPEG1:  {44100, 48000, 32000},
	MPEG2:  {22050, 24000, 16000},
	MPEG25: {11025, 12000, 8000},
}

// mpeg1Bitrates maps an MPEG-1 layer and bitrate index to a bitrate in kbit/s.
var mpeg1Bitrates = [4][15]int{
	Layer1: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
	Layer2: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
	Layer3: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
}

// lsfBitrates maps an MPEG-2 or MPEG-2.5 layer and bitrate index to a bitrate in kbit/s.
var lsfBitrates = [4][15]int{
	Layer1: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
	Layer2: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	Layer3: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
}
