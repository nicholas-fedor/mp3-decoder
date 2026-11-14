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
| `huffman/codebook`: Huffman code tables | Transcribed from FFmpeg, libavcodec/mpegaudiodec_common.c (mpa_hufflens, mpa_huffsymbols, mpa_quad_codes, mpa_quad_bits and ff_mpa_huff_data), LGPL-2.1-or-later. Every code word and length agrees with the decode trees of dist10, lsf/decoder/tables/huffdec. The 1288 entries that parse from Sieler and Sperschneider, Tables 24 to 41, agree too |
| `huffman`: decoding order of code words, linbits and signs | Sieler and Sperschneider, Table 6 (huffmancodebits) |
| `huffman`: tables 4 and 14 decoding as table 0 | FFmpeg, libavcodec/mpegaudiodec_common.c (ff_mpa_huff_data) |
| `huffman`: a count1 quadruple past the region's end left out | FFmpeg, libavcodec/mpegaudiodec_template.c (huffman_decode) |
| `layer3`: side information fields and layouts | Sieler and Sperschneider, Tables 4 and 13 and section 4.2.4 |
| `layer3`: window switching with block type 0 rejected | Sieler and Sperschneider, section 4.2.4 (block_type). FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer3). mpg123, src/libmpg123/layer3.c (III_get_side_info) |
| `layer3`: big_values above 288 read as 288 | mpg123, src/libmpg123/layer3.c (III_get_side_info) |
| `layer3`: bit reservoir of every past main data byte, up to 511, emptied on resync | mpg123, src/libmpg123/layer3.c (III_get_side_info) and src/libmpg123/parse.c |
| `layer3`: missing bytes counted and the frame's main data kept on underflow | Symphonia, symphonia-bundle-mp3/src/layer3/mod.rs (BitResevoir::fill). FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer3) |

No part adapts another decoder's code. The Huffman code tables are data transcribed from FFmpeg.
