# Design

This document explains how the decoder is built and why, part by part. Each section says what a part holds and which decoder or document it follows. Where the leading decoders differ, a table compares them.

- [Principles](#principles)
  - [Following the Leading Decoders](#following-the-leading-decoders)
  - [Tables](#tables)
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
- [Performance](#performance)

## Principles

### Following the Leading Decoders

The decoder plays what the leading decoders play. Where a written description is stricter than FFmpeg and mpg123, the decoder follows FFmpeg and mpg123. Where those two differ, each section below names the one it follows and why.

### Tables

Tables the bitstream defines are written as literals, laid out as the published tables list them. Tables derived from them are generated into `*_gen.go` files by the test of the package that uses them. `go generate` runs that test with `-update`, and without `-update` the same test fails when a file is stale. Nothing is computed when a program starts.

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

`frame.Find` takes a frame only when the header after it belongs to the same stream. A search never accepts a frame on its own header alone.

| A stream keeps | A stream may change |
| --- | --- |
| Version, layer and sampling frequency | Bitrate |
| Free format, once it has it | Padding |
| | Mode |

#### Free Format

A free-format header gives no frame size. `Find` measures it from the next two headers, up to 3456 bytes as mpg123 allows. Two headers must follow at one spacing plus each frame's padding. The returned header then carries the bitrate that spacing gives, so its `Size` works like any other. No frame at a fixed bitrate reaches 3456 bytes, so `frame.MaxSize` bounds every frame.

### Tags

#### Xing and Info Tags

`container.ParseInfo` looks for a Xing or Info tag right after the side information, where LAME writes it even in a frame with a CRC. In a protected frame it also looks after the CRC. Fields the frame ends before are left out, and the fields before them are kept, as mpg123 does.

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

| Granule | region1 starts after | region2 | Line at MPEG-1 |
| --- | --- | --- | --- |
| Without window switching | region0_count + 1 long bands | Starts region1_count + 1 long bands later | Sent |
| Short or mixed blocks | Short bands 0 to 2 of each window | None | 36 |
| Start or stop blocks | Long bands 0 to 7 | None | 36 |

For mixed blocks at the MPEG-2.5 frequencies, mpg123 starts region1 after long band 7 instead.

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

## Performance

Measured with `go test -bench` on an AMD Ryzen 9 7950X. No stage allocates.

| Stage | Input | Time |
| --- | --- | --- |
| big_values pairs | Random bits | 10 to 15 ns a pair |
| Main data from the bit reservoir | A 128 kbit/s LAME file | About 15 ns a frame |
| Scalefactors and spectral values | A 128 kbit/s LAME file | About 390 ns a granule of one channel |
| Requantization | A 128 kbit/s LAME file | About 130 ns a granule of one channel |
