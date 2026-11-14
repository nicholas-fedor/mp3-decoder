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
