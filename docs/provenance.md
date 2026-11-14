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
| `huffman`: tables 4 and 14 decoding as table 0 | FFmpeg, libavcodec/mpegaudiodec_common.c (ff_mpa_huff_data). mpg123, src/libmpg123/huffman.h (ht) |
| `huffman`: big_values pairs stopping at the end of the granule's bits, and a count1 quadruple past it left out | FFmpeg, libavcodec/mpegaudiodec_template.c (huffman_decode) |
| `layer3`: side information fields and layouts | Sieler and Sperschneider, Tables 4 and 13 and section 4.2.4 |
| `layer3`: window switching with block type 0 rejected | Sieler and Sperschneider, section 4.2.4 (block_type). FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer3). mpg123, src/libmpg123/layer3.c (III_get_side_info) |
| `layer3`: big_values above 288 read as 288 | mpg123, src/libmpg123/layer3.c (III_get_side_info) |
| `layer3`: bit reservoir of every past main data byte, up to 511, emptied on resync | mpg123, src/libmpg123/layer3.c (III_get_side_info) and src/libmpg123/parse.c |
| `layer3`: missing bytes counted and the frame's main data kept on underflow | Symphonia, symphonia-bundle-mp3/src/layer3/mod.rs (BitResevoir::fill). FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer3) |
| `layer3`: MPEG-1 scalefactor widths, order and scfsi groups | Sieler and Sperschneider, Tables 15 and 16 and section 4.2.4 (main_data) |
| `layer3`: scalefactors kept in the order sent, with scfsi reusing their places | FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer3). mpg123, src/libmpg123/layer3.c (III_get_scale_factors_1) |
| `layer3`: scalefactor band edges | Sieler and Sperschneider, Tables 42 to 59. FFmpeg, libavcodec/mpegaudiodec_common.c (ff_band_size_long and ff_band_size_short) and mpg123, src/libmpg123/l3bandgain.h (bandInfo) agree |
| `layer3`: big_values regions and the implicit regions of granules with window switching | Sieler and Sperschneider, section 4.2.4 (region0_count and region1_count). FFmpeg, libavcodec/mpegaudiodec_template.c (init_short_region and init_long_region) |
| `layer3`: spectrum decoding order and its end | Sieler and Sperschneider, section 4.3.8 (Huffman decoding). FFmpeg, libavcodec/mpegaudiodec_template.c (huffman_decode) |
| `layer3`: requantization formula | dist10, lsf/decoder/decode.c (III_dequantize_sample). Sieler and Sperschneider, section 4.3.9, whose copy lacks the equation images |
| `layer3`: pretab | Sieler and Sperschneider, Table 23. FFmpeg, libavcodec/mpegaudiodec_common.c (ff_mpa_pretab) agrees |
| `layer3`: pretab in the long bands of mixed blocks, and the mixed boundary at short band 3 | FFmpeg, libavcodec/mpegaudiodec_template.c (exponents_from_scale_factors and compute_band_indexes) |
| `layer3`: mid/side and intensity stereo formulas, the intensity bound in each window, and position 7 coding none | Sieler and Sperschneider, section 4.3.11, whose copy lacks the equation images. dist10, lsf/decoder/decode.c (III_stereo) |
| `layer3`: stereo applied before reordering, from the right channel's block type, with the bound tested on the lines and positions 8 to 15 coding none | FFmpeg, libavcodec/mpegaudiodec_template.c (compute_stereo and mp_decode_layer3). mpg123, src/libmpg123/layer3.c (III_i_stereo) shares the bound and block type |
| `layer3`: the long bands of mixed blocks coded only when no short window holds a value, and the last bands taking the position below them | dist10, lsf/decoder/decode.c (III_stereo). FFmpeg, libavcodec/mpegaudiodec_template.c (compute_stereo) and mpg123, src/libmpg123/layer3.c (III_i_stereo) agree |
| `layer3`: short block reordering | Sieler and Sperschneider, section 4.3.10. FFmpeg, libavcodec/mpegaudiodec_template.c (reorder_block) |
| `hybrid`: alias reduction butterflies and coefficients | Sieler and Sperschneider, section 4.3.12.1 and Table 60. dist10, lsf/decoder/decode.c (III_antialias) |
| `hybrid`: IMDCT, windows, overlap-add and frequency inversion | Sieler and Sperschneider, sections 4.3.12.2 to 4.3.12.5, whose copy lacks the equation images. dist10, lsf/decoder/decode.c (inv_mdct and III_hybrid) |
| `hybrid`: 4 long subbands in mixed blocks at 8 kHz | Sieler and Sperschneider, section 4.3.12.1 (sb_amount) |
| `synth`: matrixing, the V vector and the windowed sum | Sieler and Sperschneider, section 4.3.12.6 and Figure 2. dist10, lsf/decoder/decode.c (SubBandSynthesis) |
| `synth`: synthesis window taps | Transcribed from FFmpeg, libavcodec/mpegaudiodsp_data.c (ff_mpa_enwindow), LGPL-2.1-or-later, with the symmetry of libavcodec/mpegaudiodsp_template.c (ff_mpa_synth_init). Every tap agrees with dist10, lsf/decoder/tables/dewindow |
| `trig`: π and cosines | Machin's formula and the Taylor series of the cosine |

No part adapts another decoder's code. The Huffman code tables and the synthesis window taps are data transcribed from FFmpeg.
