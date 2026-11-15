# Conformance

The decoder is measured against the MPEG audio conformance streams that ISO publishes with ISO/IEC 14496-26, each decoded and compared with its 24-bit reference waveform as ISO/IEC 11172-4 clause 2.6.3 describes.

## Contents

- [Contents](#contents)
- [Streams](#streams)
- [Fetching the Streams](#fetching-the-streams)
- [Measurement](#measurement)
  - [Full Accuracy](#full-accuracy)
  - [Alignment](#alignment)
- [Results](#results)

## Streams

ISO's archive `compressedMpeg12.zip` holds 40 Layer III streams, and `referencesWav.zip` holds their references. The stream names give the version: `l3_1xxxx` streams are MPEG-1, `l3_2xxxx` streams MPEG-2 and `l3_25xxx` streams MPEG-2.5. The Layer I and Layer II streams are published only as MP4 files in `compressedMp4.zip` and are not measured.

ISO's licence forbids redistributing the streams, so they are never committed. The package `internal/conformance` lists each stream and reference with the MD5 checksum ISO publishes in `mpeg4audio-conformance2ndEd_20230125.md5`.

## Fetching the Streams

`task test` fills the cache first, then runs the tests with `MP3_DECODER_CONFORMANCE` naming it. The cache defaults to `$XDG_CACHE_HOME/mp3-decoder/conformance`, and `MP3_DECODER_CONFORMANCE` moves it. To fill a cache by hand:

```sh
go run ./internal/conformance/fetch -dir DIR
```

The fetch reads only the files it needs from ISO's archives by HTTP range requests, about 38 MB of the 9.8 GB reference archive, and checks each file's checksum before it enters the cache. A complete cache needs no network. Without `MP3_DECODER_CONFORMANCE`, the conformance test skips. With it, a missing file fails the test.

## Measurement

### Full Accuracy

Each decoded channel is compared with its reference channel, with samples between -1 and 1. A decoder has full accuracy when the RMS difference stays below 2^-15/√12, about 8.8e-6, and no difference exceeds 2^-14, about 6.1e-5.

The references are 24-bit, so even an exact decoder differs from them by their rounding. For a signal whose rounding errors spread evenly, that is an RMS of 2^-23/√12, about 3.4e-8, and a decoder at the limit of the references shows about that.

### Alignment

Most references match the decoded output from its first sample, channel for channel. A mono stream decodes into both channels alike, and its reference is compared with the left one. Some references end one frame before the decoded output, and only their samples are measured.

| Stream | Alignment |
| --- | --- |
| `l3_10201` | Runs of 10 mono, 100 stereo and 18 mono frames, each with its own reference starting at decoded samples 0, 11520 and 126720. The stereo run has two alternative references, `AA` and `noAA`, which differ over its last 19 frames, and matching either conforms |
| `l3_10501_*` | Each stream carries one or two channels of a multichannel program, and each reference holds one channel, named `f` for a front channel and `b` for a back channel. The left channel of `l3_10501_b01+b02` matches `b00` and its right channel `b01`. `l3_10501_b01_b02` is the same stream, `l3_10501_f00` is `l3_10107`, and `l3_10501_f01+f02` is `l3_10207` |

## Results

The conformance test records these results. Run it with `-update` and `MP3_DECODER_CONFORMANCE` set to rewrite them, and without `-update` it fails when any result changes.

<!-- results start -->
14 of 40 streams decode at full accuracy.

| Stream | Reference | Channel | RMS | Largest | Result |
| --- | --- | --- | --- | --- | --- |
| l3_10101 | l3_10101 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10102 | l3_10102 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10103 | l3_10103 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10104 | l3_10104 | left | 3.3e-08 | 9.8e-08 | full accuracy |
| l3_10105 | l3_10105 | left | 4.9e-04 | 9.1e-03 | outside full accuracy |
| l3_10106 | l3_10106 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10107 | l3_10107 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10201 | l3_10201_0 | left | 3.4e-08 | 5.9e-08 | full accuracy |
| l3_10201 | l3_10201_1_AA | left | 6.2e-03 | 1.5e-01 | outside full accuracy |
| l3_10201 | l3_10201_1_AA | right | 6.9e-04 | 2.6e-02 | outside full accuracy |
| l3_10201 | l3_10201_1_noAA (alternative) | left | 6.2e-03 | 1.5e-01 | outside full accuracy |
| l3_10201 | l3_10201_1_noAA (alternative) | right | 6.9e-04 | 2.6e-02 | outside full accuracy |
| l3_10201 | l3_10201_2 | left | 1.1e-02 | 1.2e-01 | outside full accuracy |
| l3_10202 | l3_10202 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10202 | l3_10202 | right | 4.1e-08 | 1.5e-07 | full accuracy |
| l3_10203 | l3_10203 | left | 1.9e-03 | 6.9e-02 | outside full accuracy |
| l3_10203 | l3_10203 | right | 7.6e-04 | 2.6e-02 | outside full accuracy |
| l3_10204 | l3_10204 | left | 9.9e-01 | 1.6e+00 | outside full accuracy |
| l3_10204 | l3_10204 | right | 9.9e-01 | 1.7e+00 | outside full accuracy |
| l3_10205 | l3_10205 | left | 3.5e-08 | 6.0e-08 | full accuracy |
| l3_10205 | l3_10205 | right | 5.0e-09 | 6.0e-08 | full accuracy |
| l3_10206 | l3_10206 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10207 | l3_10207 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10207 | l3_10207 | right | 3.4e-08 | 6.1e-08 | full accuracy |
| l3_10501_b01+b02 | l3_10501_b00 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10501_b01+b02 | l3_10501_b01 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10501_b01_b02 | l3_10501_b00 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10501_b01_b02 | l3_10501_b01 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10501_f00 | l3_10501_f00 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10501_f01+f02 | l3_10501_f01 | left | 3.4e-08 | 2.9e-07 | full accuracy |
| l3_10501_f01+f02 | l3_10501_f02 | right | 3.4e-08 | 2.0e-07 | full accuracy |
| l3_20140 | l3_20140 | | | | not decoded, 476 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20141 | l3_20141 | | | | not decoded, 476 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20142 | l3_20142 | | | | not decoded, 476 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20143 | l3_20143 | | | | not decoded, 212 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20244 | l3_20244 | | | | not decoded, 386 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20245 | l3_20245 | | | | not decoded, 822 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20246 | l3_20246 | | | | not decoded, 250 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20247 | l3_20247 | | | | not decoded, 386 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_20248 | l3_20248 | | | | not decoded, 386 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25101 | l3_25101 | | | | not decoded, 212 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25102 | l3_25102 | | | | not decoded, 136 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25103 | l3_25103 | | | | not decoded, 184 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25104 | l3_25104 | | | | not decoded, 152 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25201 | l3_25201 | | | | not decoded, 195 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25202 | l3_25202 | | | | not decoded, 195 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25203 | l3_25203 | | | | not decoded, 195 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25204 | l3_25204 | | | | not decoded, 102 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25205 | l3_25205 | | | | not decoded, 138 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25206 | l3_25206 | | | | not decoded, 144 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25207 | l3_25207 | | | | not decoded, 102 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25208 | l3_25208 | | | | not decoded, 140 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
| l3_25209 | l3_25209 | | | | not decoded, 152 frames failed: decode frame: layer3: only MPEG-1 frames are decoded |
<!-- results end -->
