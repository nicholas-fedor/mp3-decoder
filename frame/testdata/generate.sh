#!/usr/bin/env bash
# Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
# SPDX-License-Identifier: Apache-2.0
#
# Regenerates the frame fixtures in this directory.
# Requires ffmpeg and lame.

set -euo pipefail
cd "$(dirname "$0")"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

ffmpeg -hide_banner -loglevel error -y -fflags +bitexact \
	-f lavfi -i "sine=frequency=440:sample_rate=44100:duration=0.2,volume=0.5,aformat=channel_layouts=stereo" \
	-map_metadata -1 -flags:a +bitexact -c:a pcm_s16le "$tmp/tone.wav"

# CRC-protected MPEG-1 Layer III at 128 kbit/s, with no Info frame.
lame --silent -p -t -b 128 --noreplaygain "$tmp/tone.wav" protected.mp3
