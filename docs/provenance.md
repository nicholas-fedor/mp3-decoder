# Provenance

The decoder takes each part from the best source for it. Any source whose licence combines with AGPL-3.0 may be studied, and its algorithms, tables and code may be adopted. Code adapted from another decoder is listed here with its licence. The [references](references.md) list every source.

## Parts

| Part | Written from |
| --- | --- |
| `frame`: header fields and values | Sieler and Sperschneider, Tables 2 and 7 to 13 |
| `frame`: layer values | dist10, lsf/decoder/decode.c |
| `frame`: bitrate tables | dist10, lsf/decoder/common.c |
| `frame`: frame size | dist10, lsf/decoder/musicout.c. FFmpeg, libavcodec/mpegaudiodecheader.c computes the same |
| `frame`: acceptance rules | FFmpeg, libavcodec/mpegaudiodecheader.h and mpegaudiodecheader.c |
| `frame`: CRC-16 | Sieler and Sperschneider, section 4.3.3 and Table 22. dist10, lsf/decoder/common.c (update_CRC) |
| `frame`: free-format frame size | mpg123, src/libmpg123/parse.c (guess_freeformat_framesize) and the 3456-byte bound in src/libmpg123/frame.h |
| `container`: ID3v2 size | ID3v2.4. FFmpeg, libavformat/id3v2.c (ff_id3v2_match and ff_id3v2_tag_len) |
| `container`: ID3v1 and Lyrics3 v2.00 trailers | ID3v1. Lyrics3 v2.00 |
| `container`: APEv2 trailer | FFmpeg, libavformat/apetag.c (ff_ape_parse_tag) |
| `container`: Xing and Info tags | LAME info tag. FFmpeg, libavformat/mp3dec.c (mp3_parse_info_tag). mpg123, src/libmpg123/parse.c (check_lame_tag) |
| `container`: VBRI tag | FFmpeg, libavformat/mp3dec.c (mp3_parse_vbri_tag) |
| `container`: seek table offsets | LAME info tag |

No part adapts another decoder's code.
