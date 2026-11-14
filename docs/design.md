# Design

## Accepting streams

The decoder plays what the leading decoders play. Where a written description is stricter than FFmpeg and mpg123, the decoder follows FFmpeg and mpg123. A frame is still confirmed by the header that follows it before a search accepts it.

`frame.Parse` rejects only the values FFmpeg rejects: a missing sync word, the reserved version, the reserved layer, bitrate index 15 and the reserved sampling frequency. So it accepts these:

- MPEG-2.5 in all three layers, with the full MPEG-2 bitrate table. Sieler and Sperschneider, Table 8, marks MPEG-2.5 bitrate indexes 9 to 14 as forbidden.
- The reserved emphasis value.
- Free format, whose frame size the header does not give.

## Finding frames

`frame.Find` takes a frame only when the header after it belongs to the same stream. A stream keeps its version, layer and sampling frequency, and a free-format stream stays free format. Bitrate, padding and mode may change from frame to frame.

A free-format header gives no frame size. `Find` measures it from the next two headers, up to 3456 bytes as mpg123 allows. Two headers must follow at one spacing plus each frame's padding. The returned header then carries the bitrate that spacing gives, so its `Size` works like any other.

## Tags

`container.ParseInfo` looks for a Xing or Info tag right after the side information, where LAME writes it even in a frame with a CRC. In a protected frame it also looks after the CRC. Fields the frame ends before are left out, and the fields before them are kept, as mpg123 does.

The encoder delay and padding come from the LAME extension of any encoder, as mpg123 reads them. FFmpeg reads them only from LAME and FFmpeg's own encoders. The frame count excludes the tag's frame, and the byte count includes it. So `Info.Samples` is the frame count times the samples per frame, less the delay and padding.

`container.TrailerSize` removes ID3v1, APEv2 and Lyrics3 v2.00 tags from the end of a stream in any order. Taggers write APEv2 before ID3v1, so a search for APEv2 only at the very end, as FFmpeg does, would miss it.

## Huffman decoding

The code words in `huffman/codebook` are the source data. They are laid out as the published tables list them, one code word for each value, and the decoder never reads them. `huffman` decodes through lookup tables built from them, which its own test generates into `lookup_gen.go`. `go generate` runs that test with `-update`, and without `-update` the same test fails when the file is stale.

Each lookup entry is 16 bits. A leaf holds x, y and the number of the code word's bits its level takes. A pointer holds where a subtable starts and how many bits it indexes by. Each table's first level indexes by the fewest bits that send at most 1 decode in 20 to a subtable, by the probability of 2^-length a code word's length implies. That gives first levels of 3 to 10 bits and 7,337 entries in all. The entries are padded to 8,192, a power of two, so a masked index needs no bounds check. Tables that share code words share entries.

Following FFmpeg, the unused tables 4 and 14 decode as table 0, whose values are zeros coded in no bits. Following FFmpeg too, a count1 quadruple with any bit past the region's end, its signs included, is left out. mpg123 checks only the code word, so it keeps a quadruple whose signs run past the end and then reports an error.

Generated tables were measured against the same tables built at run time, with the same decode code and array type:

| | Generated | Built at run time |
| --- | --- | --- |
| Decoding table 13 from 64 KiB of random bits | 582 µs | 597 µs |
| Building every table once per process | none | 419 µs and 447 KB |

Decoding big_values pairs takes 10 to 15 ns a pair on random bits, with no allocations.

## Side information

`layer3.SideInfo` holds what the bitstream transmits, in its own terms, and nothing derived. Two fields are derived later by the stage that owns their meaning. The regions of a granule with window switching are implicit, and their boundaries depend on the scalefactor bands, so the spectrum stage sets them. The preflag of an MPEG-2 or MPEG-2.5 granule is coded in scalefac_compress, so the scalefactor stage decodes it.

A granule with window switching and block type 0 makes the frame invalid, as in FFmpeg and mpg123. A big_values count above 288 is read as 288, as mpg123 does, where FFmpeg rejects the frame.

## Bit reservoir

`layer3.Reservoir` holds the last 511 bytes of main data, the furthest a 9-bit main_data_begin reaches back. An MPEG-2 or MPEG-2.5 frame reaches back at most 255 bytes, so the same reservoir serves it. Like mpg123, the reservoir holds every byte of past main data, even bytes that granules of past frames read. FFmpeg and Symphonia hold only the bytes after the last granule they read. They lose a frame whose main data overlaps the frame before it, which mpg123 plays.

A frame's own main data is copied after the bytes it reaches back to, so later stages read one contiguous slice. No frame holds more than 3456 bytes, so a fixed buffer of 3967 bytes serves every frame and nothing is allocated. Handing out a frame's main data takes about 15 ns.

When the reservoir holds fewer bytes than main_data_begin, as at the start of a stream joined midway or after a resync, it reports how many bytes are missing. It keeps the frame's own main data either way, so the frames after it can begin in it.
