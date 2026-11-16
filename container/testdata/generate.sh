#!/usr/bin/env bash
# Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
# SPDX-License-Identifier: Apache-2.0
#
# Regenerates the container fixtures in this directory.
# Requires ffmpeg and lame. Every fixture encodes a 440 Hz sine of exactly 10000 samples.

set -euo pipefail
cd "$(dirname "$0")"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tone() {
	local rate=$1
	ffmpeg -hide_banner -loglevel error -y -fflags +bitexact \
		-f lavfi -i "sine=frequency=440:sample_rate=${rate}:duration=1,atrim=end_sample=10000,volume=0.5,aformat=channel_layouts=stereo" \
		-map_metadata -1 -flags:a +bitexact -c:a pcm_s16le "$tmp/tone-${rate}.wav"
}

tone 44100
tone 22050

# A Xing tag with the LAME extension.
lame --silent -V 2 --noreplaygain "$tmp/tone-44100.wav" vbr.mp3
# An Info tag with the LAME extension.
lame --silent -b 128 --noreplaygain "$tmp/tone-44100.wav" cbr.mp3
# An Info tag in a CRC-protected stream.
lame --silent -p -b 128 --noreplaygain "$tmp/tone-44100.wav" protected.mp3
# An Info tag in an MPEG-2 stream.
lame --silent -b 64 --noreplaygain "$tmp/tone-22050.wav" mpeg2.mp3
