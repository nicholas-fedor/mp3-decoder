#!/usr/bin/env bash
# Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
# SPDX-License-Identifier: Apache-2.0
#
# Regenerates the Layer II and Layer III fixtures in this directory and the references decoded from
# them. Requires ffmpeg, lame, twolame and mpg123. Every fixture encodes 0.5 s of a 440 Hz tone with
# a click every 0.1 s, whose sharp onsets make the encoder switch to short blocks.
#
# Each reference holds every sample mpg123 or ffmpeg decodes from a fixture as 32-bit
# little-endian floats, interleaved by channel. ffmpeg refuses free format, so free.mp3 has only
# mpg123's reference. Neither tool leaves out the encoder delay and padding,
# and both skip the Info frame. versions.txt records the tools that wrote the files.
#
# The Layer II references come from ffmpeg's floating-point decoder, mp2float. ffmpeg skips the
# first frame of layer2-joint.mp2 as junk, so that fixture has only mpg123's reference. Neither
# tool decodes free-format Layer II with the allocation table its bitrate chooses, so
# layer2-free.mp2 has no reference. It is checked against layer2-384.mp2, the same tone at a fixed
# bitrate that chooses the same table.
#
# golden.txt records the hash of the samples the decoder gives for each fixture. After regenerating
# the fixtures, rerun the decoder package's tests with -update to record their new hashes.

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
lame --silent --freeformat -b 400 --noreplaygain "$tmp/44100-stereo.wav" free.mp3

twolame --talkativity 0 --protect --mode j --bitrate 64 "$tmp/44100-stereo.wav" layer2-joint.mp2
twolame --talkativity 0 --mode m --bitrate 96 "$tmp/44100-mono.wav" layer2-mono.mp2
twolame --talkativity 0 --mode s --bitrate 64 "$tmp/22050-stereo.wav" layer2-mpeg2.mp2
twolame --talkativity 0 --freeformat --mode j --bitrate 400 "$tmp/44100-stereo.wav" layer2-free.mp2
twolame --talkativity 0 --mode j --bitrate 384 "$tmp/44100-stereo.wav" layer2-384.mp2

for name in mpeg1-joint mpeg1-mono protected; do
	mpg123 --quiet --no-gapless --encoding f32 --stdout "$name.mp3" >"$name.mpg123.f32"
	ffmpeg -hide_banner -loglevel error -y -flags2 +skip_manual -i "$name.mp3" \
		-f f32le -c:a pcm_f32le "$name.ffmpeg.f32"
done

mpg123 --quiet --no-gapless --encoding f32 --stdout free.mp3 >free.mpg123.f32

for name in layer2-joint layer2-mono layer2-mpeg2; do
	mpg123 --quiet --no-gapless --encoding f32 --stdout "$name.mp2" >"$name.mpg123.f32"
done

for name in layer2-mono layer2-mpeg2; do
	ffmpeg -hide_banner -loglevel error -y -flags2 +skip_manual -c:a mp2float -i "$name.mp2" \
		-f f32le -c:a pcm_f32le "$name.ffmpeg.f32"
done

{
	mpg123 --version | head -n 1
	ffmpeg -version | head -n 1
	lame --version | head -n 1
	twolame --help 2>&1 | grep -m 1 'TwoLAME version'
} >versions.txt
