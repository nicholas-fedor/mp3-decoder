#!/usr/bin/env bash
# Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# Regenerates the Layer III fixtures in this directory.
# Requires ffmpeg and lame. Every fixture encodes 0.5 s of a 440 Hz tone with a click every 0.1 s,
# whose sharp onsets make the encoder switch to short blocks.

set -euo pipefail
cd "$(dirname "$0")"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tone() {
	local rate=$1 layout=$2
	ffmpeg -hide_banner -loglevel error -y -fflags +bitexact \
		-f lavfi -i "aevalsrc=0.4*sin(2*PI*440*t)+0.9*lt(mod(t\,0.1)\,0.002)*sin(2*PI*3000*t):s=${rate}:d=0.5,aformat=channel_layouts=${layout}" \
		-map_metadata -1 -flags:a +bitexact -c:a pcm_s16le "$tmp/${rate}-${layout}.wav"
}

tone 44100 stereo
tone 44100 mono
tone 22050 stereo
tone 8000 mono

lame --silent -V 2 --noreplaygain "$tmp/44100-stereo.wav" mpeg1-joint.mp3
lame --silent -b 64 --noreplaygain "$tmp/44100-mono.wav" mpeg1-mono.mp3
lame --silent -b 64 --noreplaygain "$tmp/22050-stereo.wav" mpeg2.mp3
lame --silent -b 16 --noreplaygain "$tmp/8000-mono.wav" mpeg25.mp3
lame --silent -p -b 128 --noreplaygain "$tmp/44100-stereo.wav" protected.mp3
