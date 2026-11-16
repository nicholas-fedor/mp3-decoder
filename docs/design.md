# Design

This document explains how the decoder is built and why, part by part. Each section says what a part holds and which decoder or document it follows. Where the leading decoders differ, a table compares them.

- [Principles](#principles)
  - [Following the Leading Decoders](#following-the-leading-decoders)
  - [Tables](#tables)
  - [Determinism](#determinism)
- [Stream Structure](#stream-structure)
  - [Frame Headers](#frame-headers)
  - [Finding Frames](#finding-frames)
    - [Free Format](#free-format)
  - [Tags](#tags)
    - [Xing and Info Tags](#xing-and-info-tags)
    - [Encoder Delay and Padding](#encoder-delay-and-padding)
    - [End-of-Stream Tags](#end-of-stream-tags)
- [Layer III Decoding](#layer-iii-decoding)
  - [Side Information](#side-information)
    - [Fields](#fields)
    - [Invalid Values](#invalid-values)
  - [Bit Reservoir](#bit-reservoir)
    - [What It Holds](#what-it-holds)
    - [Underflow](#underflow)
  - [Scalefactors](#scalefactors)
    - [Storage Order](#storage-order)
    - [scfsi After Short Blocks](#scfsi-after-short-blocks)
    - [Lower Sampling Frequencies](#lower-sampling-frequencies)
  - [Huffman Decoding](#huffman-decoding)
    - [Code Tables](#code-tables)
    - [Lookup Tables](#lookup-tables)
    - [Where Decoding Stops](#where-decoding-stops)
  - [Scalefactor Bands](#scalefactor-bands)
    - [Edges](#edges)
    - [Sources](#sources)
  - [Spectrum](#spectrum)
    - [Regions](#regions)
  - [Requantization](#requantization)
    - [Formula](#formula)
    - [Mixed Blocks](#mixed-blocks)
    - [Exact Tables](#exact-tables)
  - [Stereo](#stereo)
    - [Mid/Side](#midside)
    - [Intensity Bound](#intensity-bound)
    - [Intensity Positions](#intensity-positions)
    - [Lower Sampling Frequencies](#lower-sampling-frequencies-1)
  - [Reordering](#reordering)
  - [Hybrid Filterbank](#hybrid-filterbank)
    - [Stages](#stages)
    - [Fast IMDCT](#fast-imdct)
    - [Mixed Blocks at 8 kHz](#mixed-blocks-at-8-khz)
    - [Mixed Start and Stop Blocks](#mixed-start-and-stop-blocks)
    - [Channel Changes](#channel-changes)
    - [Exact Transform Tables](#exact-transform-tables)
  - [Polyphase Synthesis](#polyphase-synthesis)
    - [Flow](#flow)
    - [Fast Matrixing](#fast-matrixing)
    - [Window](#window)
- [Layer I Decoding](#layer-i-decoding)
  - [Bit Allocation](#bit-allocation)
  - [Scalefactor Table](#scalefactor-table)
  - [Sample Requantization](#sample-requantization)
  - [Values No Encoder Writes](#values-no-encoder-writes)
- [Layer II Decoding](#layer-ii-decoding)
  - [Allocation Tables](#allocation-tables)
  - [Choosing a Table](#choosing-a-table)
  - [Quantization Classes](#quantization-classes)
  - [Frame Layout](#frame-layout)
  - [Scalefactor Selection](#scalefactor-selection)
  - [Words No Encoder Writes](#words-no-encoder-writes)
- [Stream Decoding](#stream-decoding)
  - [Opening a Stream](#opening-a-stream)
  - [Length](#length)
  - [Frame Decoding](#frame-decoding)
  - [Missing Main Data](#missing-main-data)
  - [Tags and Damage Between Frames](#tags-and-damage-between-frames)
  - [End of a Stream](#end-of-a-stream)
  - [Gapless Playback](#gapless-playback)
  - [Seeking](#seeking)
  - [Damage Counts](#damage-counts)
  - [Accuracy](#accuracy)
- [Performance](#performance)
  - [Measured Changes](#measured-changes)

## Principles

### Following the Leading Decoders

The decoder plays what the leading decoders play. Where a written description is stricter than FFmpeg and mpg123, the decoder follows FFmpeg and mpg123. Where those two differ, each section below names the one it follows and why.

### Tables

Tables the bitstream defines are written as literals, laid out as the published tables list them. Tables derived from them are generated into `*_gen.go` files by the test of the package that uses them. `go generate` runs that test with `-update`, and without `-update` the same test fails when a file is stale. Nothing is computed when a program starts.

Generated values are computed from closed forms with `math/big` and rounded to the nearest float64, so no table depends on the accuracy of package `math`. Sines and cosines come from the test-only package `internal/trig`, which sums their Taylor series to 320 bits with π from Machin's formula.

### Determinism

Every machine decodes the same bits. The Go specification lets a compiler fuse a product with a sum into one multiply-add, which rounds once where the code rounds twice. Only an explicit conversion to a floating-point type forbids it. arm64 fuses such pairs by default. amd64 fuses them at GOAMD64=v3, and only when both operands sit in registers. So every floating-point product is converted where it is made, as in `float64(a * b)`. `TestProductsRounded` type-checks the decoder's production code and fails on any product that is not.

`TestGolden` records the SHA-256 hash of the samples each fixture decodes to, and the [conformance results](conformance.md#results) record one hash of the samples of all the conformance streams. `task test:fma` reruns the decoder's tests at GOAMD64=v3, and on arm64 under qemu-aarch64. With one synthesis product left unconverted, the arm64 run changed both hashes. The GOAMD64=v3 run changed neither, since that product reads a table from memory.

## Stream Structure

### Frame Headers

`frame.Parse` rejects only the values FFmpeg rejects:

- A missing sync word.
- The reserved version and the reserved layer.
- Bitrate index 15.
- The reserved sampling frequency.

So it accepts these, which a stricter reading would reject:

- MPEG-2.5 in all three layers, with the full MPEG-2 bitrate table. Sieler and Sperschneider, Table 8, marks MPEG-2.5 bitrate indexes 9 to 14 as forbidden.
- The reserved emphasis value.
- Free format, whose frame size the header does not give.

### Finding Frames

`frame.Find` takes a frame only when the header after it belongs to the same stream. A search never accepts a frame on its own header alone. Once the stream has ended, a header whose frame would reach past its end can never be confirmed, so the decoder's search moves past it to the frames after it.

| A stream keeps | A stream may change |
| --- | --- |
| Version, layer and sampling frequency | Bitrate |
| Free format, once it has it | Padding |
| | Mode |

#### Free Format

A free-format header gives no frame size. `Find` measures it from the next two headers, up to 3456 bytes as mpg123 allows. Two headers must follow at one spacing plus each frame's padding. The returned header then carries the bitrate that spacing gives, so its `Size` works like any other. No frame at a fixed bitrate reaches 3456 bytes, so `frame.MaxSize` bounds every frame.

The decoder gives every frame of a free-format stream the bitrate measured at its first frame. mpg123 decodes free format, though in Layer II it takes the wrong allocation table, as [Choosing a Table](#choosing-a-table) shows. FFmpeg n9.0.2 refuses to open either free-format fixture.

### Tags

#### Xing and Info Tags

`container.ParseInfo` looks for a Xing or Info tag right after the side information, where LAME writes it even in a frame with a CRC. In a protected frame it also looks after the CRC. Fields the frame ends before are left out, and the fields before them are kept, as mpg123 does.

Only a Layer III frame holds a tag. mpg123 reads the LAME tag only in Layer III, and FFmpeg's tag reader gives up on any other layer. Nor do the Layer II encoders write one. TwoLAME has no tag writer, and FFmpeg's MP3 muxer refuses Layer II audio, so its MP2 output holds bare frames. So the first frame of a Layer I or Layer II stream is audio whatever its bytes hold, and the decoder delay of 529 samples, which belongs to Layer III, never applies to those layers.

#### Encoder Delay and Padding

The encoder delay and padding come from the LAME extension of any encoder, as mpg123 reads them. FFmpeg reads them only from LAME and FFmpeg's own encoders. The frame count excludes the tag's frame, and the byte count includes it. So `Info.Samples` is the frame count times the samples per frame, less the delay and padding.

#### End-of-Stream Tags

`container.TrailerSize` removes ID3v1, APEv2 and Lyrics3 v2.00 tags from the end of a stream in any order. Taggers write APEv2 before ID3v1, so a search for APEv2 only at the very end, as FFmpeg does, would miss it.

## Layer III Decoding

The package `internal/layer3` decodes each granule of each channel in these stages:

1. [Side information](#side-information) gives the fields of every granule.
2. The [bit reservoir](#bit-reservoir) hands the frame its main data.
3. [Scalefactors](#scalefactors) come first in each granule's part of the main data.
4. [Spectral values](#spectrum) follow, Huffman coded in regions bounded by the [scalefactor bands](#scalefactor-bands).
5. [Requantization](#requantization) scales the spectral values by the gains and scalefactors.
6. [Stereo](#stereo) restores the left and right channels of a joint stereo granule.
7. [Reordering](#reordering) interleaves the windows of short blocks for the IMDCT.
8. The [hybrid filterbank](#hybrid-filterbank), in the package `internal/hybrid`, turns each channel's lines into subband samples.
9. [Polyphase synthesis](#polyphase-synthesis), in the package `internal/synth`, turns each set of 32 subband samples into 32 PCM samples.

### Side Information

#### Fields

`layer3.SideInfo` holds what the bitstream transmits, in its own terms, and nothing derived. Two fields are derived later by the stage that owns their meaning:

- The regions of a granule with window switching are implicit, and their boundaries depend on the scalefactor bands, so the [spectrum](#regions) stage sets them.
- The preflag of an MPEG-2 or MPEG-2.5 granule is coded in scalefac_compress, so the scalefactor stage decodes it.

#### Invalid Values

| Value | This decoder | FFmpeg | mpg123 |
| --- | --- | --- | --- |
| Window switching with block type 0 | Rejects the frame | Rejects the frame | Rejects the frame |
| big_values above 288 | Reads 288 | Rejects the frame | Reads 288 |

### Bit Reservoir

#### What It Holds

`layer3.Reservoir` holds the last 511 bytes of main data, the furthest a 9-bit main_data_begin reaches back. An MPEG-2 or MPEG-2.5 frame reaches back at most 255 bytes, so the same reservoir serves it.

Like mpg123, the reservoir holds every byte of past main data, even bytes that granules of past frames read. FFmpeg and Symphonia hold only the bytes after the last granule they read. They lose a frame whose main data overlaps the frame before it, which mpg123 plays.

A frame's own main data is copied after the bytes it reaches back to, so later stages read one contiguous slice. A fixed buffer of 3967 bytes, the reservoir plus `frame.MaxSize`, serves every frame, so nothing is allocated.

#### Underflow

When the reservoir holds fewer bytes than main_data_begin, as at the start of a stream joined midway or after a resync, it reports how many bytes are missing. It keeps the frame's own main data either way, so the frames after it can begin in it.

### Scalefactors

#### Storage Order

`layer3.Scalefactors` keeps a granule's scalefactors in the order the bitstream sends them, and `Long` and `Short` find them by band. Both granules of a channel are read into one value, so the second granule keeps the groups that scfsi marks without a copy. The values stay as sent, without the pretab, since the second granule reuses them under its own preflag.

#### scfsi After Short Blocks

A stream should never mark groups when either granule of the channel has short blocks. One that does reuses the values sent in the same places, as FFmpeg and mpg123 both read it.

#### Lower Sampling Frequencies

MPEG-2 and MPEG-2.5 frames send their scalefactors in four partitions. The 9-bit scalefac_compress gives each partition's width as the digits of a mixed-radix number, and picks one of six rows of partition counts, which FFmpeg and mpg123 hold alike. Values from 500 up also set preflag, which these frames do not send. The right channel of an intensity stereo frame halves scalefac_compress, whose lowest bit is the intensity scale, and takes rows of its own. `lsfSplit` holds each range's first value, radices and row, so one function reads every range.

Mixed blocks at these frequencies send long bands 0 to 5, which end at line 36 like the 8 long bands of MPEG-1, and then short bands 3 to 11. `Scalefactors` keeps how far short values sit before their places in short blocks, 1 in MPEG-1 and 3 here.

### Huffman Decoding

#### Code Tables

The code words in `huffman/codebook` are the source data. They are laid out as the published tables list them, one code word for each value, and the decoder never reads them. `huffman` decodes through lookup tables built from them, which its own test generates into `lookup_gen.go`.

#### Lookup Tables

Each lookup entry is 16 bits:

- A leaf holds x, y and the number of the code word's bits its level takes.
- A pointer holds where a subtable starts and how many bits it indexes by.

Each table's first level indexes by the fewest bits that send at most 1 decode in 20 to a subtable, by the probability of 2^-length a code word's length implies. That gives first levels of 3 to 10 bits and 7,337 entries in all. The entries are padded to 8,192, a power of two, so a masked index needs no bounds check. Tables that share code words share entries.

Generated tables were measured against the same tables built at run time, with the same decode code and array type:

| | Generated | Built at run time |
| --- | --- | --- |
| Decoding table 13 from 64 KiB of random bits | 582 µs | 597 µs |
| Building every table once per process | none | 419 µs and 447 KB |

#### Where Decoding Stops

| Case | This decoder, as FFmpeg | mpg123 |
| --- | --- | --- |
| Unused tables 4 and 14 | Decode as table 0, zeros coded in no bits | Decodes as table 0 |
| big_values pairs that reach the end of the granule's bits | Stop there, decoding whole a pair that starts before the end | Decodes every pair, then drops the frame |
| A count1 quadruple with any bit past the end, its signs included | Left out | Kept when its code word ends in time, then an error |

### Scalefactor Bands

#### Edges

Each sampling frequency has one entry that holds its rate, the first line of each long band and the first line of each short band within a window. Edges serve every stage that uses them. Region boundaries index them directly, and a band's lines run from one edge to the next. The edges are `int`, so they index the values without conversions. One frequency's entry takes 296 bytes, and a stream uses only one.

#### Sources

FFmpeg, mpg123 and Sieler and Sperschneider agree on every edge at all nine frequencies. dist10 starts long band 18 at 24 kHz on line 330, where the others start it on line 332, and mpg123 records the change as a fixed bug.

### Spectrum

`layer3.Spectrum` holds a granule's 576 quantized values and how many the Huffman data gave. The big_values pairs come in up to three regions, then count1 quadruples until the granule's bits end or fewer than 4 lines remain. The [Huffman rules](#where-decoding-stops) decide where a damaged granule stops.

#### Regions

A granule without window switching sends its region boundaries as counts of long bands. A granule with window switching has implicit boundaries, set as FFmpeg sets them:

| Granule | region1 starts after | region2 | Line at MPEG-1 | Line at MPEG-2 and 11.025 or 12 kHz | Line at 8 kHz |
| --- | --- | --- | --- | --- | --- |
| Without window switching | region0_count + 1 long bands | Starts region1_count + 1 long bands later | Sent | Sent | Sent |
| Short or mixed blocks | Short bands 0 to 2 of each window | None | 36 | 36 | 72 |
| Start or stop blocks | Long bands 0 to 7 | None | 36 | 54 | 108 |

The conformance stream `l3_25102` holds short blocks at 8 kHz and decodes at full accuracy from line 72. For mixed blocks at the MPEG-2.5 frequencies, mpg123 starts region1 after long band 7 instead, line 54 at 11.025 and 12 kHz. No conformance stream holds them.

### Requantization

#### Formula

`Spectrum.Requantize` turns each quantized value v into sign(v) |v|^(4/3) 2^(e/4), as dist10 computes it. The exponent e counts quarter steps of gain:

| Part of e | Long bands | Short bands |
| --- | --- | --- |
| Global gain | global_gain - 210 | global_gain - 210 |
| Window | | Less 8 for each step of the window's subblock_gain |
| Scalefactor | Less the scalefactor plus, with preflag, the pretab | Less the scalefactor |
| Steps of a scalefactor | 2, or 4 with scalefac_scale | 2, or 4 with scalefac_scale |

`Scalefactors` keeps each granule's preflag, so requantization reads it in one place for every version. Lines past the last decoded value are 0 without being computed.

#### Mixed Blocks

Mixed blocks scale their lines as long bands up to where short band 3 starts, and as short bands from there, as FFmpeg does. That boundary is line 36 at MPEG-1 and MPEG-2, and line 72 at 8 kHz, where no short band starts at line 36. The same boundary starts [region1](#regions) of mixed blocks.

#### Exact Tables

|v|^(4/3) comes from a table of every magnitude from 0 to 8206, and 2^(e/4) from one of the four powers 2^(r/4) times a power of two. Both tables are generated from closed forms computed to 256 bits with `math/big`, then rounded to the nearest float64. The power of two is built from its exponent bits, which is exact for every gain the side information can give, from 2^-325 to 2^11. So no transcendental function runs while decoding, and the result is the same on every architecture.

Building the power of two from its bits takes about 125 ns a granule against 155 ns with `math.Ldexp`, for the same values.

### Stereo

`layer3.Stereo` restores the left and right channels of a joint stereo granule from the requantized lines, before reordering, as FFmpeg, Symphonia and dist10 order the stages. A stream should give both channels the same block type. Like FFmpeg and mpg123, the stage reads the block type and intensity positions of the right channel.

#### Mid/Side

Mid/side stereo turns the mid and side lines M and S into left (M+S)/√2 and right (M-S)/√2. With intensity stereo on too, only the bands that intensity stereo does not code take it.

#### Intensity Bound

Intensity stereo codes the bands above the highest band in which the right channel holds a value. The stage tests the requantized lines, as FFmpeg and mpg123 do, rather than the count of decoded values. So decoded zeros above the last value lie above the bound, and a silent band below that value does not.

| Granule | Bound |
| --- | --- |
| Long blocks | The highest long band with a value |
| Short blocks | The highest short band with a value, in each window |
| Mixed blocks | Each window's short bands as in short blocks. The long bands have a bound of their own only when no window holds a value in a short band, and take none otherwise |

The long bands of mixed blocks end where short band 3 starts, as in [requantization](#mixed-blocks).

#### Intensity Positions

A band above the bound holds the left channel's value v and takes its position p from the right channel's scalefactors. It becomes left v k/(1+k) and right v/(1+k), where k = tan(p π/12). The ratios for positions 0 to 6 are generated from their closed forms in √3, computed to 256 bits with `math/big`, as the [requantization tables](#exact-tables) are.

| Case | This decoder, as FFmpeg | mpg123 | dist10 |
| --- | --- | --- | --- |
| Position 7 | Codes none | Codes none | Codes none |
| Positions 8 to 15, which 4-bit scalefactors in long bands 0 to 10 and short bands 0 to 5 can send | Code none | Scale by tan(p π/12), which makes position 9 scale by about 10^38 | Scale by tan(p π/12) |
| Long band 21 and short band 12, which send no scalefactor | Take the position of the band below | Take the position of the band below | Take the position of the band below |

A band that codes no position takes mid/side stereo when the frame has it, and keeps its channels otherwise.

#### Lower Sampling Frequencies

MPEG-2 and MPEG-2.5 frames take other ratios. The lowest bit of the right channel's scalefac_compress, the intensity scale, gives a ratio r of 2^(-1/4), or 2^(-1/2) when it is set. Position 0 keeps the left channel's value v in both channels, an odd position p scales the left channel to v r^((p+1)/2), and an even one scales the right channel to v r^(p/2). The ratios come from the exact powers of 2^(1/4) that requantization uses.

Which position codes none differs between decoders, and the conformance streams settle it:

| Rule | Decoder | `l3_20247` left RMS | Streams with intensity stereo at full accuracy |
| --- | --- | --- | --- |
| Position 7 codes none | mpg123 | 3.8e-2 | 2 of 11 |
| No position codes none | FFmpeg | 3.8e-2 | 2 of 11 |
| The largest value a scalefactor's width holds, 2^slen - 1, codes none | This decoder | 3.4e-8 | 7 of 11, and all 11 with the rule for the highest bands below |

So `Scalefactors` keeps the largest value each scalefactor's width holds.

Long band 21 and short band 12 send no position. The streams show that they take the position of the band below only when intensity stereo codes that band too, and position 0 otherwise. In `l3_20245` only band 21 lies above the bound in some frames, and taking band 20's position there scales the left channel by 2^(-3/4) where the reference keeps it. With that rule, every stream with intensity stereo decodes at full accuracy. MPEG-1 keeps the position of the band below in either case, as FFmpeg and mpg123 do, and no MPEG-1 conformance stream has only its highest band above the bound.

### Reordering

Short blocks hold each short band's lines window by window, as the bitstream sends them. `layer3.Reorder` moves line f of window w, counted from the window's first line, to line 3f+w. The 18 lines of each subband then hold its 6 frequencies, the three windows interleaved, which is how the IMDCT reads them. Mixed blocks keep their long block lines in place.

FFmpeg, Symphonia and dist10 reorder in a pass of their own after stereo, and mpg123 writes the requantized lines in this order directly. A separate pass keeps stereo working on whole runs of one window. Short blocks are rare, so the pass costs little.

### Hybrid Filterbank

`hybrid.Filterbank` turns one channel's granule of 576 lines into 18 samples of each of the 32 polyphase subbands. It keeps the second half of each subband's last block for the next granule. The caller says how many of the lowest subbands hold long blocks, and which window they take, so the package needs no Layer III types.

#### Stages

| Stage | What it does |
| --- | --- |
| Alias reduction | 8 butterflies across each boundary between two subbands of long blocks. A butterfly turns the lower line l and upper line u into l Cs - u Ca and u Cs + l Ca, where Cs = 1/√(1+C²) and Ca = C/√(1+C²) |
| Long blocks | A 36-point IMDCT of each subband's 18 lines, then the normal, start or stop window |
| Short blocks | Three 12-point IMDCTs of 6 lines each, short windowed and starting 6, 12 and 18 samples into the 36-sample block |
| Overlap-add | The first half of each block adds to the second half of the subband's block before |
| Frequency inversion | Every odd sample of every odd subband is negated |

#### Fast IMDCT

An IMDCT of 2n lines gives 4n samples, 36 from 18 for long blocks and 12 from 6 for short blocks. Sample i is the sum over the lines X(k) of X(k) cos(π/(8n) (2i+1+2n)(2k+1)). Those angles fold onto the DCT-IV of the lines, c(j), the sum over k of X(k) cos(π/(8n) (2j+1)(2k+1)):

| Samples | Value |
| --- | --- |
| i below n | c(n+i) |
| i from n to 3n-1 | -c(3n-1-i) |
| i from 3n on | -c(i-3n) |

The DCT-IV comes from a DCT-II of the same length. Since 2 cos(α) cos(β) is cos(β-α) + cos(β+α), the lines scaled by 1/(2 cos(π(2k+1)/(8n))) have a DCT-II D with c(j) = D(j) + D(j+1), where D(2n) is 0. Byeong Gi Lee's split, as in [Fast Matrixing](#fast-matrixing), turns that DCT-II into two of n points, 9 for long blocks and 3 for short blocks, which are summed directly. Each of those folds its values in pairs, since value k and value n-1-k meet the same cosine, the second with the sign (-1)^m. An even output sums the pairs' sums and adds or takes the middle value, and an odd output sums the pairs' differences. A long block takes 99 products instead of 648, and a short window 15 instead of 72. The direct sums stay in the tests as the oracles.

The kernels `imdct36` and `imdct12` run as straight-line code that the tests generate into `imdct_gen.go` from a loop form of the same steps, and they check that the kernels give its samples bit for bit. The loop form never adds a value to a 0 it starts from, since +0 plus -0 is +0, so the unrolled kernels give the same signs of 0.

The scaling reaches 11.5 for long blocks, so the fast form rounds a little more. Against an exact sum, the largest error of a long block is 7.7e-15 and of a short block 3.2e-15. That is about 4 million times below the 24-bit noise floor, and no conformance stream's RMS or maximum difference changed in its recorded digits.

#### Mixed Blocks at 8 kHz

The long blocks of mixed blocks end where short band 3 starts, line 36 or line 72 at 8 kHz, as in [requantization](#mixed-blocks). So the filterbank transforms 2 subbands as long blocks, or 4 at 8 kHz, and reduces aliasing between them.

| Decoder | Long subbands of mixed blocks at 8 kHz |
| --- | --- |
| This decoder | 4, as Sieler and Sperschneider give for alias reduction |
| FFmpeg | 2, marked unchecked in its code, though its requantization and reordering treat 72 lines as long |
| mpg123 | 2 |
| dist10 | 2 |

With 2, lines 36 to 71 would take short block IMDCTs while every stage before them holds those lines as long block lines in order of frequency.

#### Mixed Start and Stop Blocks

The mixed block flag may be set on a start or stop block, where every stage before the filterbank holds the granule as long blocks. The subbands that would hold the long part of a mixed block, 2, or 4 at 8 kHz, then take the normal window, and the rest take the start or stop window. FFmpeg, mpg123 and dist10 window those subbands so, and the conformance streams `l3_10105`, `l3_10201` and `l3_10203` match only so. `hybrid.Block` carries the count as `Mixed`.

#### Channel Changes

A stream may change between mono and stereo from one frame to the next. Both channels' filterbanks, hybrid and polyphase, then start afresh, as at the start of a stream, while the bit reservoir carries on. `synth.Filterbanks` keeps this rule for every layer, and tells Layer III when to restart the hybrid filterbanks too. ISO's references for `l3_10201`, which changes from mono to stereo and back, start each run from silent filterbanks, and match only so. FFmpeg and mpg123 carry the left channel's filterbanks over the change, and carrying them over differs from the references by up to 0.15 in the first granule after it.

#### Exact Transform Tables

The alias butterflies, the windows and the IMDCT's factors and cosines are generated into `filterbank_gen.go`. Every angle is a multiple of π/72, and the published coefficients C are parsed as exact decimals. A forward MDCT in the tests encodes random subband signals through long, start, short and stop blocks, and the filterbank gives them back to within 1e-12.

### Polyphase Synthesis

`synth.Filterbank` is the polyphase synthesis filterbank that every layer shares. Each call turns one sample of each of the 32 subbands into the next 32 PCM samples, unclipped.

#### Flow

The filterbank follows the published flow directly:

1. Matrixing turns the subband samples S(k) into 64 values V(i), the sum over k of S(k) cos(π/64 (16+i)(2k+1)).
2. V becomes the newest of the 16 vectors kept, and the oldest is dropped.
3. Output sample j sums 16 window taps. For each of the 8 groups g, it takes tap D(64g+j) times value j of vector 2g, and tap D(64g+32+j) times value 32+j of vector 2g+1, counting vectors from the newest.

The 16 vectors live in a ring of 1024 values, so a call writes 64 values instead of shifting all 1024. The ring is kept twice over, so the window reads each group's values in one run without wrapping around.

#### Fast Matrixing

Matrixing computes V from a 32-point DCT of the subband samples, X(m), the sum over k of S(k) cos(π/64 m(2k+1)). The angles of V fold onto those of X by the cosine's symmetries:

| V(i) | Value |
| --- | --- |
| i from 0 to 15 | X(16+i) |
| i of 16 | 0 |
| i from 17 to 47 | -X(48-i) |
| i from 48 to 63 | -X(i-48) |

Byeong Gi Lee's algorithm computes the DCT. It splits n values into the sums x(k) + x(n-1-k) and the differences x(k) - x(n-1-k), each difference scaled by 1/(2 cos(π(2k+1)/(2n))). Two DCTs of n/2 points follow. The even outputs are those of the first. Each odd output adds two neighboring outputs of the second. Down to 2 points, that takes 80 products instead of the 2048 of the direct sum. The direct sum stays in the tests as the oracle. The DCT runs as straight-line code that the tests generate from the recursion into `dct_gen.go`, and they check that it gives the recursion's outputs bit for bit.

The scaling reaches 10.2 at 32 points, so the fast form rounds a little more than the direct sum. The largest error of an impulse response rose from 6.7e-16 to 1.3e-15. That is 25 million times below the 24-bit noise floor, and no conformance stream's RMS or maximum difference changed in its recorded digits.

#### Window

The 512 taps D(i) are exact multiples of 2^-16. The window is generated from PDMP3's table of the taps to 9 decimals, with each tap rounded to the nearest multiple. The test checks that every published tap lies within its rounding of that multiple. It also checks that tap 512-i is tap i, negated unless i is a multiple of 64.

Two further checks bind the window and the matrixing together:

- The impulse response of subband k is D(n) cos(π/64 (16 + n mod 64)(2k+1)) for the 512 samples the window spans, then silence.
- Random PCM through the published analysis filterbank, whose window is D/32, and back through synthesis returns 481 samples late. The error stays under 1e-4, the near reconstruction the filterbank pair was designed for.

## Layer I Decoding

The package `internal/layer12` decodes a Layer I frame in these stages:

1. The [bit allocation](#bit-allocation) gives the width of each subband's samples.
2. A scalefactor follows for each subband with samples, from the [scalefactor table](#scalefactor-table).
3. Twelve sets of samples follow, each with one sample of every subband, which [requantization](#sample-requantization) scales.
4. [Polyphase synthesis](#polyphase-synthesis) turns each set into 32 PCM samples, as in Layer III.

A Layer I frame stands alone. Nothing but the synthesis carries over from one frame to the next, so `layer12.Decoder` keeps only each channel's synthesis filterbank.

### Bit Allocation

Each subband carries a 4-bit allocation for each channel. An allocation n from 1 to 14 gives samples of n+1 bits, and 0 gives the subband no samples. In joint stereo, the subbands from a bound on carry one allocation and one sample for both channels, and each channel scales the sample by its own scalefactor. The mode extension sets the bound to subband 4, 8, 12 or 16. In every other mode the bound is 32, so each channel keeps its own samples throughout.

### Scalefactor Table

A 6-bit index selects each scalefactor, 2^(1-i/3) for index i. The table is generated into `scalefactors_gen.go`, each entry the cube root of 2^(3-i) rounded to the nearest float64, so every third entry is an exact power of two. Layer II shares the table.

### Sample Requantization

A sample of n bits with code c stands for 2(c+1-2^(n-1))/(2^n-1) times its scalefactor. The codes divide the range from -1 to 1 into 2^n-1 steps, centred on 0. The decoder works out the step of each subband once a frame, as the scalefactor times 2/(2^n-1), and multiplies each sample's offset from the centre by it. A Layer I sample of n bits is a class of 2^n-1 steps, and `quantClass` requantizes the samples of both layers by this one rule.

### Values No Encoder Writes

Two Layer I values are never written. The allocation 15 is forbidden, since it could pass for a sync word. The scalefactor index 63 is left out of the table. The leading decoders treat them so:

| Value | mpg123 | libmad | FFmpeg | This decoder |
| --- | --- | --- | --- | --- |
| Allocation 15 | Aborts the frame, which then gives silence | Fails the frame | Reads samples of 16 bits | Silences the frame |
| Scalefactor index 63 | Scales the subband by 0 | Scales the subband by 0, and fails the frame only in a strict build | Continues the table to 2^-20 | Scales the subband by 0 |

The decoder follows mpg123 and libmad, which agree. A frame with the forbidden allocation counts as `Undecodable`, as in Layer III.

## Layer II Decoding

### Allocation Tables

A Layer II frame codes only the lower subbands, and an allocation table says how. For each subband it gives the width of the subband's allocation, which is 2, 3 or 4 bits. It also gives the quantization class that each allocation from 1 selects. MPEG-1 has four tables, of 27, 30, 8 and 12 subbands, and the lower sampling frequencies share a fifth of 30. `allocation.go` holds them as runs of subbands that code alike, with each class named for its steps. FFmpeg, mpg123 and libmad hold the same five tables, entry for entry.

### Choosing a Table

An MPEG-1 frame's table follows its sampling frequency and its bitrate per channel. ISO lists the bitrates that choose each table, and the leading decoders choose alike at every fixed bitrate. They part only at free-format bitrates between those the lists name:

| Bitrate per channel | mpg123 | FFmpeg | libmad | This decoder |
| --- | --- | --- | --- | --- |
| Fixed bitrates | As ISO lists | As ISO lists | As ISO lists | As ISO lists |
| Free format, up to 48 kbit/s | 27 subbands | 8 subbands, or 12 at 32 kHz | 27 at 48 kHz, 30 otherwise | 8 subbands, or 12 at 32 kHz |
| Free format, above 48 and below 56 kbit/s | 27 subbands | 12 subbands | 27 at 48 kHz, 30 otherwise | 8 subbands, or 12 at 32 kHz |
| Free format, 56 to 80 kbit/s | 27 subbands | 27 subbands | 27 at 48 kHz, 30 otherwise | 27 subbands |
| Free format, above 80 and below 96 kbit/s | 27 subbands | 27 at 48 kHz, 12 otherwise | 27 at 48 kHz, 30 otherwise | 27 at 48 kHz, 30 otherwise |
| Free format, from 96 kbit/s | 27 subbands | 27 at 48 kHz, 30 otherwise | 27 at 48 kHz, 30 otherwise | 27 at 48 kHz, 30 otherwise |

mpg123 chooses by the bitrate index, so a free-format stream always takes its entry for index 0, and decodes free-format streams from TwoLAME wrongly. FFmpeg and libmad choose by the bitrate a free-format stream measures. libmad takes the listed tables only at exactly the bitrates the lists name. Between them FFmpeg falls through to the table of 12 subbands, which ISO gives only to 32 kHz. This decoder reads ISO's lists as ranges, so every bitrate falls in one. Below 56 kbit/s a frame takes the table of 8 subbands, or 12 at 32 kHz, as the lowest bitrates do. Up to 80 kbit/s it takes the table of 27. Above that it takes 27 at 48 kHz and 30 otherwise, as the highest bitrates do.

TwoLAME chooses the table of a free-format stream by its bitrate too. Its stream at 400 kbit/s takes the table of 30 subbands, and decodes to within an RMS of 8.0e-4 of the same tone at a fixed 384 kbit/s, which takes the same table. mpg123's output of it differs from that by an RMS of 0.48. `TestFreeFormatLayer2` checks the decoder against the fixed-rate stream.

### Quantization Classes

A class gives the steps of its samples and how they are coded. The classes of 3, 5 and 9 steps code three samples in one code word of 5, 7 or 10 bits. The first sample takes the lowest place. Every other class has 2^n-1 steps and codes each sample in n bits, from 3 bits for 7 steps to 16 bits for 65535. A code k of a class of N steps stands for (2k-(N-1))/N times its scalefactor, so the steps centre on 0 and fill the range from -1 to 1.

### Frame Layout

A Layer II frame holds 36 sets of subband samples in three parts of 12. Each part has its own scalefactors, and its samples come in four granules of three sets. The frame carries these fields in turn:

1. The allocation of each subband the table codes, for each channel below the joint stereo bound and once for both from it on. The bound may lie above the subbands the table codes, and then every subband the table codes is below it.
2. A scalefactor selection of 2 bits for each subband and channel with samples.
3. The scalefactors each selection calls for.
4. Each granule's codes, subband by subband, for each channel below the bound and once for both from it on. Both channels scale shared codes by their own scalefactors.

`layer12.Decoder` decodes the three sets of a granule, then synthesizes them, so it keeps only those sets. The subbands above those the table codes hold no samples.

### Scalefactor Selection

A selection says which parts share a subband's scalefactors, and so how many the frame carries:

| Selection | Scalefactors carried | Part 1 | Part 2 | Part 3 |
| --- | --- | --- | --- | --- |
| 0 | 3 | The first | The second | The third |
| 1 | 2 | The first | The first | The second |
| 2 | 1 | The first | The first | The first |
| 3 | 2 | The first | The second | The second |

FFmpeg, mpg123 and libmad read the selections so.

### Words No Encoder Writes

A grouped code word of n bits can hold more than the N³ words three samples of N steps need. The classes of 3 and 5 steps leave 5 and 3 words unused, and the class of 9 steps leaves 295. No encoder writes them, and the leading decoders read them differently:

| Decoder | A grouped word past N³ |
| --- | --- |
| mpg123 | Its grouping tables hold 0 past the valid words, so the three samples are 0 |
| libmad | Takes each code modulo N, so the third sample wraps |
| FFmpeg | Leaves the third sample unbounded, past the class's steps |
| This decoder | Gives the three samples 0, as mpg123 does |

Silence for a word no encoder writes matches the scalefactor index 63 in both layers. Layer II has no forbidden allocation, so every Layer II frame decodes.

## Stream Decoding

### Opening a Stream

`mp3.OpenStream` moves past the ID3v2, ID3v1 and APEv2 tags at the start of a stream, then searches for the first frame that the header after it confirms, as [Finding Frames](#finding-frames) describes. That frame's header fixes what the rest of the stream must keep. A Xing or Info tag frame that opens the stream holds no audio, so it is skipped, as mpg123 and FFmpeg skip it.

A stream may be opened at any byte, as on joining a radio stream. Decoding starts at the first confirmed frame, and every frame from there gives its samples. The frames whose main data began before the join give silence, as [Missing Main Data](#missing-main-data) describes. A frame reaches back at most 511 bytes in MPEG-1 and 255 at the lower sampling frequencies, so only the first few frames can. The filterbanks then take up to two granules to hold only what came after the silence. From there the samples match a decode of the whole stream exactly. A first frame needs the header after it to confirm it, and a free-format one needs two, so a stream opened within its last frame or two finds no frame.

### Length

`mp3.Open` takes a file rather than a stream, and decodes only the bytes before the tags at its end. It also finds how many samples the file decodes to, which `Len` reports. A tag frame that counts both the stream's frames and its bytes gives the length, when the byte count is exactly the size of the audio from the tag frame to the tags at the end. Otherwise a walk over the frames counts them as decoding would take them, through the same search for frames, which reads the whole file.

FFmpeg drops the frame count only when the file is larger than the byte count by more than a sixteenth, which it takes for files joined together, and keeps it otherwise. `Len` must be exact, so this decoder takes the count only on an exact match. Every fixture's tag matches, so they open without a walk, and a fixture joined to a copy of itself is walked.

The walk also fills the frame index that [Seeking](#seeking) uses. `mp3.OpenStream` knows no length.

### Frame Decoding

`layer3.Decoder` decodes a Layer III frame through every stage, granule by granule. It keeps what one frame passes to the next: the bit reservoir, the scalefactors that scfsi reuses, and each channel's filterbanks. `layer12.Decoder` decodes a Layer I or Layer II frame. A stream keeps its layer, so the decoder of the first frame's layer decodes them all. A mono frame gives each sample to both channels, and samples are not clipped.

A frame that holds a value no encoder writes fails to decode. In Layer III that is a value of its side information, and in Layer I it is the forbidden allocation. It gives silence in its place, as mpg123 fills such a frame with zeros. Every frame then keeps its samples, so `Len`, positions and seeks stay exact after damage.

The hybrid filterbank takes the block of each granule from Layer III: every subband long, with the normal, start or stop window, for long blocks, none for short blocks, and those below where short band 3 starts for mixed blocks.

### Missing Main Data

A frame whose main data begins before the bytes the reservoir holds gives silence in both granules, as mpg123 handles it. This happens on joining a stream midway, where the first frames reach back to main data that was never received. The frame's own main data still enters the reservoir, and the filterbanks still run on the silent granules, so the frames after it continue from the right state.

The leading decoders differ here. FFmpeg and Symphonia silence only the granules whose main data is missing, and decode a second granule whose main data the reservoir holds. ISO's reference decoder produces nothing for such a frame, so its references start with the first frame whose main data is all there, and the conformance stream `l3_10204`, which starts midway, matches only when that frame starts from silent filterbanks. Silencing the whole frame matches the reference from its first frame on, while keeping the output's length and timing.

### Tags and Damage Between Frames

After each frame, the decoder expects the next frame of the stream. It steps over an ID3v2 tag, an ID3v1 tag or an APEv2 tag that starts with its header, as mpg123 does. A stream joined from several tracks carries their tags between its frames. Stepping over a tag keeps the bit reservoir, so tags change no sample.

Any other bytes are damage or junk. The decoder passes over them to the next frame that the header after it confirms, and empties the bit reservoir, as mpg123 does on a resync. The bytes passed over may hold main data that the next frames reach back to, and decoding a frame from the wrong main data gives noise. Instead, a frame whose main data began before the junk gives silence, as [Missing Main Data](#missing-main-data) describes. On the joint stereo fixture, junk before a frame changes the samples of that frame and the three after it. A damaged header loses its own frame and changes the samples of the three after it.

### End of a Stream

A stream ends where the source ends, or where a frame of another stream starts. Such a frame has another version, layer or sampling frequency. mpg123 plays on across that change unless it is told not to. `Read` hands out samples at one rate, so the stream ends there. A last frame that the stream ends inside is decoded from the bytes it holds, with zeros for the rest, as FFmpeg does, since its granules' main data often lies wholly in the bytes it holds or in the frames before it. A last frame cut inside its side information ends the stream before it.

### Gapless Playback

An encoder adds samples at the start of a stream, its delay, and pads the last frame. The decoder's filterbanks delay its output by a further 529 samples. When the tag frame's LAME extension gives the delay and padding, the decoder leaves out the samples they account for, so a track decodes to exactly the samples its encoder took in, and the tracks of an album play without gaps.

| Samples left out | mpg123 | FFmpeg | This decoder |
| --- | --- | --- | --- |
| At the start | The delay plus 529 | The delay plus 529 | The delay plus 529 |
| At the end, with the tag's frame count | Every sample from the padding less 529 before the counted frames end, since decoding stops there | The padding less 529, before the counted frames end | The padding less 529, before the counted frames end |
| At the end, without a frame count | None, and none at the start either | None | None |

The decoder follows FFmpeg, whose frames past the count still play, so a stream that runs on past its tag's count loses no audio. mpg123's gapless output of each tagged fixture is exactly the samples of every frame less those, and each fixture decodes to exactly the 0.5 s its encoder took in.

### Seeking

`Seek` moves a file that `mp3.Open` opened to any sample, and the samples from there are those of a decode from the start, bit for bit. A frame's samples depend on more than its own bytes, in two ways:

- Its main data may begin up to 511 bytes back in MPEG-1, and 255 bytes back at the lower sampling frequencies.
- The hybrid filterbank's overlap and the polyphase synthesis carry samples of each granule into the next two.

So decoding starts afresh enough frames earlier to cover both. In Layers I and II only the synthesis reaches back. It carries each set of 32 subband samples into the output of the next 15. A Layer I frame holds 12 sets, so decoding starts 2 frames before the target's frame. A Layer II frame holds 36, so it starts 1 frame before. In Layer III the reservoir sets the count. The first frames must fill the bit reservoir that far even when they are the smallest frames the stream could hold. Those are frames at its lowest bitrate with a CRC and two channels. The frames of the two granules before the target's frame follow. Each stage keeps only this bounded past. The synthesis also sums its values in the same order wherever its ring buffer stands. So from the target's frame on, the state is the one a decode from the start reaches. The frames decoded first may give silence, and their samples are dropped. `frame.Header.Preroll` gives the count from a stream's header, so a player that opens a file partway through, at a byte offset, can start that many frames before the sample it wants too. On a damaged stream, a frame in that span that fails to decode adds no main data to the reservoir, so the frames just after it may decode otherwise than from the start.

| Stream | Smallest frame's main data | Frames for the reservoir | Frames for the filterbanks | Frames decoded first |
| --- | --- | --- | --- | --- |
| MPEG-1 at 32 kHz | 106 bytes | 5 | 1 | 6 |
| MPEG-1 at 44.1 kHz | 66 bytes | 8 | 1 | 9 |
| MPEG-1 at 48 kHz | 58 bytes | 9 | 1 | 10 |
| MPEG-2 at 22.05 kHz | 3 bytes | 85 | 2 | 87 |
| MPEG-2 at 24 kHz | 1 byte | 255 | 2 | 257 |
| MPEG-2.5 at 8 kHz | 49 bytes | 6 | 2 | 8 |
| Layer I | No reservoir | 0 | 2 | 2 |
| Layer II | No reservoir | 0 | 1 | 1 |

The bound needs no side information, so the frame to start from follows from frame numbers alone. mpg123 decodes 4 frames before the target by default, which suits most streams but can fall short of the reservoir's reach. On the fixtures, starting only 3 frames back already changes samples of the joint stereo and MPEG-2 fixtures.

Frames are found through an index of where the search for each one starts, as mpg123 indexes them. It holds up to 1,024 entries. When it fills, it drops every other entry and doubles its step. The walk in `mp3.Open` fills it, and so does decoding. A seek past the frames indexed walks on from the last entry. A stream that `mp3.OpenStream` opened cannot seek.

### Damage Counts

Damage never ends a stream, and `Stats` counts how often decoding carried on past each kind of it:

| Count | What it counts | What the frame gives |
| --- | --- | --- |
| `Frames` | Frames decoded | Its samples |
| `Resyncs` | Passes over bytes that are not a frame of the stream | The frames after it, as [Tags and Damage Between Frames](#tags-and-damage-between-frames) describes |
| `CRCMismatches` | Frames whose CRC does not match | Its samples, decoded all the same |
| `Underflows` | Frames whose main data begins before the bytes the reservoir holds | Silence, as [Missing Main Data](#missing-main-data) describes |
| `Undecodable` | Frames that hold a value no encoder writes | Silence |

The CRC covers the last two bytes of a frame's header and the first bits after the CRC, which each layer's decoder checks, since only it knows how many there are:

| Layer | Bits after the CRC that it covers |
| --- | --- |
| Layer I | The bit allocation, 4 bits for each subband and channel below the bound and for each subband from it on |
| Layer II | The bit allocation, then the scalefactor selection of each subband and channel with samples |
| Layer III | The side information |

A frame whose CRC does not match is decoded anyway, as FFmpeg and mpg123 decode it. mpg123 never checks the CRC. FFmpeg checks it only when its `err_detect` option asks, and then only logs a mismatch unless `explode` is set too. In a stereo Layer I frame, FFmpeg's check covers 256 bits whatever the bound, which in joint stereo runs past the allocation. libmad covers the allocation alone, as this decoder does. libmad and FFmpeg cover the same Layer II bits. Every frame of the fixtures LAME and TwoLAME encoded with CRCs passes, and so does every frame of the conformance streams, which carry CRCs in seven Layer I streams and fourteen Layer II streams. A damaged CRC changes no sample. The frames a seek decodes before its sample are not counted, since the first of them give silence by design.

### Accuracy

The fixtures decode against the 32-bit float output of mpg123 1.33.7 and FFmpeg n9.0.2, with neither leaving out the encoder delay. FFmpeg's Layer II output comes from its floating-point decoder, mp2float, since its default MP2 decoder computes in fixed point. FFmpeg skips the first frame of the joint stereo Layer II fixture as junk, so it is measured from the 480th sample after it, where FFmpeg's fresh synthesis holds only its own frames:

| Fixture | RMS difference from mpg123 | Largest difference from mpg123 | RMS difference from FFmpeg | Largest difference from FFmpeg |
| --- | --- | --- | --- | --- |
| VBR joint stereo, LAME -V 2 | 5.1e-8 | 5.1e-7 | 1.1e-7 | 1.1e-6 |
| 64 kbit/s mono | 7.7e-8 | 7.1e-7 | 1.4e-7 | 1.1e-6 |
| 128 kbit/s stereo with CRCs | 4.5e-8 | 3.0e-7 | 1.0e-7 | 1.1e-6 |
| 400 kbit/s free format | 4.7e-8 | 3.2e-7 | Not decoded | Not decoded |
| Layer II, 64 kbit/s joint stereo with CRCs | 2.2e-8 | 2.0e-7 | 1.1e-7 past its first frame | 9.9e-7 past its first frame |
| Layer II, 96 kbit/s mono | 3.3e-8 | 5.3e-7 | 1.7e-7 | 1.5e-6 |
| Layer II, MPEG-2 at 64 kbit/s | 2.4e-8 | 2.5e-7 | 2.2e-7 | 2.4e-6 |

Full accuracy allows an RMS difference of 8.8e-6 and a largest difference of 6.1e-5, so each fixture sits about 100 times inside it. The difference from mpg123 is at the rounding of its 32-bit output. FFmpeg's float decoder computes in 32 bits, which doubles its difference.

The ISO conformance streams are measured against their 24-bit references, as [Conformance](conformance.md) records.

## Performance

Measured with `go test -bench` on an AMD Ryzen 9 7950X. The joint stereo fixture is 0.5 s of a 440 Hz tone with clicks, which LAME encoded at -V 2. No stage allocates, and `TestAllocations` checks that `Read` and `Seek` allocate nothing once a stream has started.

| Stage | Input | Time |
| --- | --- | --- |
| big_values pairs | Random bits | 10 to 15 ns a pair |
| Main data from the bit reservoir | The joint stereo fixture | About 15 ns a frame |
| Scalefactors and spectral values | The joint stereo fixture | About 390 ns a granule of one channel |
| Requantization | The joint stereo fixture | About 130 ns a granule of one channel |
| Stereo | The joint stereo fixture | About 230 ns a granule of both channels |
| Reordering | A granule of short blocks | About 310 ns |
| Hybrid filterbank, with the fast IMDCT | Random lines | About 1.8 µs a granule of long blocks, and 1.7 µs of short blocks |
| Polyphase synthesis, with the fast DCT | Random subband samples | About 3.3 µs a granule of one channel |
| A frame through every stage | The joint stereo fixture | About 24 µs a frame |
| Opening and decoding a whole stream | The joint stereo fixture | About 890 times faster than real time |
| A seek, then reading the frame there | The joint stereo fixture, indexed | About 265 µs, mostly the 9 frames decoded before the sample |
| A Layer I frame through every stage | Random stereo frames with samples in every subband | About 7.5 µs a frame of 384 samples |
| A Layer II frame through every stage | Random stereo frames at 192 kbit/s | About 26 µs a frame of 1152 samples |

### Measured Changes

Each change to the transforms was measured on its own, on the joint stereo fixture. A change that keeps the order of every sum keeps the output bit for bit, and the others were recorded with new hashes.

| Change | Measured effect | Kept |
| --- | --- | --- |
| Synthesis matrixing through Lee's 32-point DCT | Synthesis from 19 µs to 7.0 µs a granule | Yes |
| IMDCT through a DCT-IV, a DCT-II and Lee's split | Long blocks from 5.9 µs to 3.2 µs, short blocks from 3.8 µs to 2.9 µs | Yes |
| Synthesis ring kept twice over, read without wrapping | Synthesis from 7.0 µs to 5.5 µs, the same bits | Yes |
| The 32-point DCT generated as straight-line code | Synthesis from 5.5 µs to 3.3 µs, the same bits | Yes |
| The DCTs of the IMDCT folded in pairs | Long blocks from 3.1 µs to 2.9 µs, short blocks slower | Yes, for the kernels below |
| The IMDCTs generated as straight-line code | Long blocks from 2.9 µs to 1.8 µs, short blocks from 3.0 µs to 1.7 µs | Yes |
| Synthesis windowing with each sample's sum in a register | Synthesis from 3.3 µs to 4.3 µs, since each addition waits on the one before | No |
| Skipping the IMDCT of subbands whose lines are all 0 | Not built. Its gain depends on how many subbands are silent. The fixtures, a tone with clicks, leave about 21 of 32 silent, far more than music, so they cannot measure it fairly | No |

The synthesis window now takes about 37% of decoding. It runs 32 independent sums near one multiply-add a cycle, the most scalar Go gives.
