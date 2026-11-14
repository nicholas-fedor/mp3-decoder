# Design

## Accepting streams

The decoder plays what the leading decoders play. Where a written description is stricter than FFmpeg and mpg123, the decoder follows FFmpeg and mpg123. A frame is still confirmed by the header that follows it before a search accepts it.

`frame.Parse` rejects only the values FFmpeg rejects: a missing sync word, the reserved version, the reserved layer, bitrate index 15 and the reserved sampling frequency. So it accepts these:

- MPEG-2.5 in all three layers, with the full MPEG-2 bitrate table. Sieler and Sperschneider, Table 8, marks MPEG-2.5 bitrate indexes 9 to 14 as forbidden.
- The reserved emphasis value.
- Free format, whose frame size the header does not give.
