# Conformance

The decoder is measured against the MPEG audio conformance streams that ISO publishes with ISO/IEC 14496-26, each decoded and compared with its 24-bit reference waveform as ISO/IEC 11172-4 clause 2.6.3 describes.

## Contents

- [Contents](#contents)
- [Streams](#streams)
- [Fetching the Streams](#fetching-the-streams)
- [Measurement](#measurement)
  - [Full Accuracy](#full-accuracy)
  - [Alignment](#alignment)
  - [Exact Output](#exact-output)
- [Results](#results)
- [Census](#census)

## Streams

ISO's archive `compressedMpeg12.zip` holds 40 Layer III streams, and `referencesWav.zip` holds their references. The stream names give the version. `l3_1xxxx` streams are MPEG-1, `l3_2xxxx` streams are MPEG-2, and `l3_25xxx` streams are MPEG-2.5. The archive `compressedMp4.zip` holds the 26 Layer I and Layer II streams, each in an MP4 file that stores one frame a sample. The conformance test reads the frames out of each file's sample table. `l1_` streams are Layer I and `l2_` streams are Layer II, apart from `l2_test33`, which holds MPEG-2 Layer I frames. `l2_test27` has no reference, so 25 of them are measured.

ISO's licence forbids redistributing the streams, so they are never committed. The package `internal/conformance` lists each stream and reference with the MD5 checksum ISO publishes in `mpeg4audio-conformance2ndEd_20230125.md5`.

## Fetching the Streams

`task test` fills the cache first, then runs the tests with `MP3_DECODER_CONFORMANCE` naming it. The cache defaults to `$XDG_CACHE_HOME/mp3-decoder/conformance`, and `MP3_DECODER_CONFORMANCE` moves it. To fill a cache by hand:

```sh
go run ./internal/conformance/fetch -dir DIR
```

The fetch reads only the files it needs from ISO's archives by HTTP range requests, about 43 MB of the 1.2 GB MP4 archive and the 9.8 GB reference archive, and checks each file's checksum before it enters the cache. A complete cache needs no network. Without `MP3_DECODER_CONFORMANCE`, the conformance test skips. With it, a missing file fails the test.

## Measurement

### Full Accuracy

Each decoded channel is compared with its reference channel, with samples between -1 and 1. A decoder has full accuracy when the RMS difference stays below 2^-15/√12, about 8.8e-6, and no difference exceeds 2^-14, about 6.1e-5.

The references are 24-bit, so even an exact decoder differs from them by their rounding. For a signal whose rounding errors spread evenly, that is an RMS of 2^-23/√12, about 3.4e-8, and a decoder at the limit of the references shows about that.

### Alignment

Most references match the decoded output from its first sample, channel for channel. A mono stream decodes into both channels alike, and its reference is compared with the left one. Some references end one frame before the decoded output, and only their samples are measured. Decoded samples are clipped to the range a 24-bit reference holds before they are compared, since `l3_10204` decodes past full scale and its reference holds the clipped samples.

| Stream | Alignment |
| --- | --- |
| `l3_10201` | Runs of 10 mono, 100 stereo and 18 mono frames, each with its own reference starting at decoded samples 0, 11520 and 126720. The stereo run has two alternative references, `AA` and `noAA`, which differ over its last 19 frames, and matching either conforms |
| `l3_10204` | The stream starts midway, and the main data of its first 2 frames lies before its start. The decoder gives those frames as silence, while the reference leaves them out, so it starts at decoded sample 2304 |
| `l3_10501_*` | Each stream carries one or two channels of a multichannel program, and each reference holds one channel, named `f` for a front channel and `b` for a back channel. The left channel of `l3_10501_b01+b02` matches `b00` and its right channel `b01`. `l3_10501_b01_b02` is the same stream, `l3_10501_f00` is `l3_10107`, and `l3_10501_f01+f02` is `l3_10207` |

### Exact Output

The test also hashes the samples it decodes from each stream, and the results record one SHA-256 hash of those hashes, stream by stream. Every machine must decode the same bits, so the hash checks the machines that fuse multiply-adds, which `task test:fma` tests on.

## Results

The conformance test records these results and the census below. Run it with `-update` and `MP3_DECODER_CONFORMANCE` set to rewrite them, and without `-update` it fails when any result changes.

<!-- results start -->
49 of 65 streams decode at full accuracy.

The decoded samples of all 65 streams hash to `66bbe77dfca80ccb348eaff56790a4fecffb87fae5db95e9d5145d5df9a4da33`.

| Stream | Reference | Channel | RMS | Largest | Result |
| --- | --- | --- | --- | --- | --- |
| l1_fl1 | l1_fl1 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl1 | l1_fl1 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl2 | l1_fl2 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl2 | l1_fl2 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl3 | l1_fl3 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl3 | l1_fl3 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl4 | l1_fl4 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl5 | l1_fl5 | left | 3.5e-08 | 6.0e-08 | full accuracy |
| l1_fl5 | l1_fl5 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl6 | l1_fl6 | left | 3.5e-08 | 6.0e-08 | full accuracy |
| l1_fl6 | l1_fl6 | right | 3.5e-08 | 6.0e-08 | full accuracy |
| l1_fl7 | l1_fl7 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl7 | l1_fl7 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl8 | l1_fl8 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l1_fl8 | l1_fl8 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l2_fl10 | l2_fl10 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl11 | l2_fl11 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl12 | l2_fl12 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl13 | l2_fl13 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl14 | l2_fl14 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl15 | l2_fl15 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_fl16 | l2_fl16 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test24 | l2_test24 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test25 | l2_test25 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test26 | l2_test26 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test28 | l2_test28 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test29 | l2_test29 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test30 | l2_test30 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test31 | l2_test31 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test32 | l2_test32 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l2_test33 | l2_test33 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l2_test33 | l2_test33 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l2_test34 | l2_test34 | | | | not opened, with the error "mp3: Layer II streams are not decoded" |
| l3_10101 | l3_10101 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10102 | l3_10102 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10103 | l3_10103 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10104 | l3_10104 | left | 3.3e-08 | 9.8e-08 | full accuracy |
| l3_10105 | l3_10105 | left | 3.5e-08 | 6.5e-08 | full accuracy |
| l3_10106 | l3_10106 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10107 | l3_10107 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10201 | l3_10201_0 | left | 3.4e-08 | 5.9e-08 | full accuracy |
| l3_10201 | l3_10201_1_AA | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10201 | l3_10201_1_AA | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_10201 | l3_10201_1_noAA (alternative) | left | 3.6e-05 | 1.3e-04 | outside full accuracy |
| l3_10201 | l3_10201_1_noAA (alternative) | right | 5.4e-05 | 3.1e-04 | outside full accuracy |
| l3_10201 | l3_10201_2 | left | 3.3e-08 | 6.0e-08 | full accuracy |
| l3_10202 | l3_10202 | left | 3.5e-08 | 8.7e-08 | full accuracy |
| l3_10202 | l3_10202 | right | 4.1e-08 | 1.5e-07 | full accuracy |
| l3_10203 | l3_10203 | left | 3.0e-08 | 6.0e-08 | full accuracy |
| l3_10203 | l3_10203 | right | 3.5e-08 | 6.0e-08 | full accuracy |
| l3_10204 | l3_10204 | left | 3.4e-08 | 6.2e-08 | full accuracy |
| l3_10204 | l3_10204 | right | 3.4e-08 | 6.2e-08 | full accuracy |
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
| l3_20140 | l3_20140 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20141 | l3_20141 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20142 | l3_20142 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20143 | l3_20143 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20244 | l3_20244 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20244 | l3_20244 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20245 | l3_20245 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20245 | l3_20245 | right | 3.4e-08 | 6.1e-08 | full accuracy |
| l3_20246 | l3_20246 | left | 3.2e-08 | 6.0e-08 | full accuracy |
| l3_20246 | l3_20246 | right | 3.4e-08 | 6.1e-08 | full accuracy |
| l3_20247 | l3_20247 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20247 | l3_20247 | right | 2.1e-08 | 6.0e-08 | full accuracy |
| l3_20248 | l3_20248 | left | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_20248 | l3_20248 | right | 3.4e-08 | 6.0e-08 | full accuracy |
| l3_25101 | l3_25101 | left | 5.5e-08 | 2.7e-07 | full accuracy |
| l3_25102 | l3_25102 | left | 6.1e-08 | 2.7e-07 | full accuracy |
| l3_25103 | l3_25103 | left | 7.1e-08 | 4.0e-07 | full accuracy |
| l3_25104 | l3_25104 | left | 6.7e-08 | 4.6e-07 | full accuracy |
| l3_25201 | l3_25201 | left | 6.0e-08 | 2.8e-07 | full accuracy |
| l3_25201 | l3_25201 | right | 6.0e-08 | 3.1e-07 | full accuracy |
| l3_25202 | l3_25202 | left | 5.6e-08 | 2.2e-07 | full accuracy |
| l3_25202 | l3_25202 | right | 0.0e+00 | 0.0e+00 | full accuracy |
| l3_25203 | l3_25203 | left | 5.6e-08 | 2.4e-07 | full accuracy |
| l3_25203 | l3_25203 | right | 5.4e-08 | 1.2e-07 | full accuracy |
| l3_25204 | l3_25204 | left | 6.6e-08 | 4.2e-07 | full accuracy |
| l3_25204 | l3_25204 | right | 5.6e-08 | 2.2e-07 | full accuracy |
| l3_25205 | l3_25205 | left | 6.4e-08 | 4.3e-07 | full accuracy |
| l3_25205 | l3_25205 | right | 6.4e-08 | 4.3e-07 | full accuracy |
| l3_25206 | l3_25206 | left | 6.5e-08 | 3.4e-07 | full accuracy |
| l3_25206 | l3_25206 | right | 6.5e-08 | 3.4e-07 | full accuracy |
| l3_25207 | l3_25207 | left | 1.2e-07 | 1.2e-06 | full accuracy |
| l3_25207 | l3_25207 | right | 1.1e-07 | 1.0e-06 | full accuracy |
| l3_25208 | l3_25208 | left | 1.3e-07 | 1.2e-06 | full accuracy |
| l3_25208 | l3_25208 | right | 1.3e-07 | 1.3e-06 | full accuracy |
| l3_25209 | l3_25209 | left | 1.2e-07 | 1.1e-06 | full accuracy |
| l3_25209 | l3_25209 | right | 1.2e-07 | 1.2e-06 | full accuracy |
<!-- results end -->

## Census

The census lists the features each stream uses, read from its frame headers and side information. A feature no stream uses is tested only by the decoder's own tests.

<!-- census start -->
| Feature | Streams |
| --- | --- |
| Layer I | l1_fl1, l1_fl2, l1_fl3, l1_fl4, l1_fl5, l1_fl6, l1_fl7, l1_fl8, l2_test33 |
| Layer II | l2_fl10, l2_fl11, l2_fl12, l2_fl13, l2_fl14, l2_fl15, l2_fl16, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test30, l2_test31, l2_test32, l2_test34 |
| Layer III | l3_10101, l3_10102, l3_10103, l3_10104, l3_10105, l3_10106, l3_10107, l3_10201, l3_10202, l3_10203, l3_10204, l3_10205, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20143, l3_20244, l3_20245, l3_20246, l3_20247, l3_20248, l3_25101, l3_25102, l3_25103, l3_25104, l3_25201, l3_25202, l3_25203, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| MPEG-1 | l1_fl1, l1_fl2, l1_fl3, l1_fl4, l1_fl5, l1_fl6, l1_fl7, l1_fl8, l2_fl10, l2_fl11, l2_fl12, l2_fl13, l2_fl14, l2_fl15, l2_fl16, l3_10101, l3_10102, l3_10103, l3_10104, l3_10105, l3_10106, l3_10107, l3_10201, l3_10202, l3_10203, l3_10204, l3_10205, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02 |
| MPEG-2 | l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test30, l2_test31, l2_test32, l2_test33, l2_test34, l3_20140, l3_20141, l3_20142, l3_20143, l3_20244, l3_20245, l3_20246, l3_20247, l3_20248 |
| MPEG-2.5 | l3_25101, l3_25102, l3_25103, l3_25104, l3_25201, l3_25202, l3_25203, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| 8 kHz | l3_25102, l3_25204, l3_25207 |
| 11.025 kHz | l3_25103, l3_25201, l3_25202, l3_25203, l3_25205, l3_25208 |
| 12 kHz | l3_25101, l3_25104, l3_25206, l3_25209 |
| 16 kHz | l2_test24, l3_20140 |
| 22.05 kHz | l2_test25, l2_test33, l3_20141, l3_20244, l3_20245, l3_20246, l3_20247, l3_20248 |
| 24 kHz | l2_test26, l2_test28, l2_test29, l2_test30, l2_test31, l2_test32, l2_test34, l3_20142, l3_20143 |
| 32 kHz | l1_fl1, l1_fl4, l2_fl10, l2_fl13, l3_10101 |
| 44.1 kHz | l1_fl2, l1_fl6, l1_fl7, l1_fl8, l2_fl11, l3_10102, l3_10104, l3_10105, l3_10106, l3_10201, l3_10202, l3_10203, l3_10204, l3_10205 |
| 48 kHz | l1_fl3, l1_fl5, l2_fl12, l2_fl14, l2_fl15, l2_fl16, l3_10103, l3_10107, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02 |
| bitrate changes | l3_10101, l3_10102, l3_10103, l3_20140, l3_20141, l3_20142, l3_20245, l3_25102, l3_25103, l3_25104, l3_25204, l3_25205, l3_25206 |
| free format | none |
| CRC | l1_fl1, l1_fl2, l1_fl3, l1_fl5, l1_fl6, l1_fl7, l2_fl10, l2_fl11, l2_fl12, l2_fl14, l2_fl15, l2_fl16, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test30, l2_test31, l2_test33, l2_test34, l3_10202 |
| mono | l1_fl4, l2_fl13, l2_test30, l3_10101, l3_10102, l3_10103, l3_10104, l3_10105, l3_10106, l3_10107, l3_10201, l3_10206, l3_10501_f00, l3_20140, l3_20141, l3_20142, l3_20143, l3_25101, l3_25102, l3_25103, l3_25104 |
| stereo | l1_fl1, l1_fl2, l1_fl3, l1_fl6, l1_fl7, l1_fl8, l2_fl10, l2_fl11, l2_fl12, l2_fl15, l2_fl16, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test32, l2_test33, l2_test34, l3_10201, l3_10202 |
| dual channel | l1_fl5, l2_fl14, l2_test31, l3_10201 |
| joint stereo, neither | l3_10201, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f01+f02, l3_20244, l3_20245, l3_20248, l3_25201, l3_25202, l3_25203, l3_25207, l3_25208, l3_25209 |
| joint stereo, mid/side | l3_10201, l3_10204, l3_10205, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f01+f02, l3_20246 |
| joint stereo, intensity | l3_10201, l3_10203, l3_20244, l3_20245, l3_20247, l3_20248, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| joint stereo, mid/side and intensity | l3_10201, l3_10203, l3_20246, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| joint stereo from subband 4 | l1_fl1, l1_fl2, l1_fl3, l2_fl10, l2_fl11, l2_fl12, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test33 |
| joint stereo from subband 8 | l1_fl1, l1_fl2, l1_fl3, l2_fl10, l2_fl11, l2_fl12, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test33 |
| joint stereo from subband 12 | l1_fl1, l1_fl2, l1_fl3, l2_fl10, l2_fl11, l2_fl12, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test33 |
| joint stereo from subband 16 | l1_fl1, l1_fl2, l1_fl3, l2_fl10, l2_fl11, l2_fl12, l2_test24, l2_test25, l2_test26, l2_test28, l2_test29, l2_test33 |
| a change between mono and stereo | l3_10201 |
| normal blocks | l3_10101, l3_10102, l3_10103, l3_10104, l3_10105, l3_10106, l3_10107, l3_10201, l3_10202, l3_10203, l3_10204, l3_10205, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20143, l3_20244, l3_20245, l3_20246, l3_20247, l3_20248, l3_25101, l3_25102, l3_25103, l3_25104, l3_25201, l3_25202, l3_25203, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| start blocks | l3_10105, l3_10107, l3_10201, l3_10203, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20244, l3_25102, l3_25103, l3_25104, l3_25201, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| short blocks | l3_10105, l3_10107, l3_10201, l3_10203, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20244, l3_20245, l3_20246, l3_25102, l3_25103, l3_25104, l3_25201, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| stop blocks | l3_10105, l3_10107, l3_10201, l3_10203, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20244, l3_25102, l3_25103, l3_25104, l3_25201, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| mixed short blocks | l3_10105, l3_10201, l3_10203 |
| mixed start or stop blocks | l3_10105, l3_10201, l3_10203 |
| scfsi | l3_10104, l3_10107, l3_10204, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02 |
| preflag bit of MPEG-1 | l3_10104, l3_10107, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02 |
| scalefac_scale | l3_10104, l3_10105, l3_10107, l3_10204, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20143, l3_20245, l3_20246, l3_20247, l3_20248, l3_25101, l3_25102, l3_25103, l3_25104, l3_25202, l3_25203, l3_25204, l3_25207, l3_25208, l3_25209 |
| big_values tables with linbits | l3_10101, l3_10102, l3_10103, l3_10104, l3_10105, l3_10106, l3_10107, l3_10201, l3_10202, l3_10203, l3_10204, l3_10205, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20143, l3_20245, l3_20246, l3_20247, l3_20248, l3_25101, l3_25102, l3_25103, l3_25104, l3_25202, l3_25203, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| count1 table B | l3_10106, l3_10107, l3_10204, l3_10206, l3_10207, l3_10501_b01+b02, l3_10501_b01_b02, l3_10501_f00, l3_10501_f01+f02, l3_20140, l3_20141, l3_20142, l3_20143, l3_20244, l3_20245, l3_20246, l3_20247, l3_20248, l3_25101, l3_25102, l3_25103, l3_25104, l3_25201, l3_25202, l3_25203, l3_25204, l3_25205, l3_25206, l3_25207, l3_25208, l3_25209 |
| main data before the stream starts | l3_10204 |
| a last frame cut short | l3_10204, l3_10206 |
<!-- census end -->
