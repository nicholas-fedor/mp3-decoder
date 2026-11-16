# Provenance

The decoder takes each part from the best source for it. Any source may be studied, and its algorithms and behavior adopted. Code and data are adapted only from sources in the public domain or under a permissive licence. Code adapted from another decoder is listed here with its licence. The [references](references.md) list every source.

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
| `container`: tags at the start of bytes, an APEv2 tag by its header | mpg123, src/libmpg123/parse.c (wetwork and handle_apetag) |
| `container`: Xing and Info tags | LAME info tag. FFmpeg, libavformat/mp3dec.c (mp3_parse_info_tag). mpg123, src/libmpg123/parse.c (check_lame_tag) |
| `container`: tags read only in Layer III frames | mpg123, src/libmpg123/parse.c (check_lame_tag, called only for Layer III). FFmpeg, libavformat/mp3dec.c (mp3_parse_vbr_tags) |
| `container`: VBRI tag | FFmpeg, libavformat/mp3dec.c (mp3_parse_vbri_tag) |
| `container`: seek table offsets | LAME info tag |
| `huffman/codebook`: Huffman code tables | The ISO code words, in this decoder's layout. Every code word and length agrees with the decode trees of PDMP3, pdmp3.c (g_huffman_table), and dist10, lsf/decoder/tables/huffdec. The 1288 entries that parse from Sieler and Sperschneider, Tables 24 to 41, agree too |
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
| `hybrid`: the IMDCT through a DCT-IV, from a DCT-II split by Byeong Gi Lee's algorithm | The folding onto the DCT-IV and the DCT-II with neighboring outputs added follow from the cosine's symmetries and product-to-sum identity. B. G. Lee 1984, in the form of Project Nayuki's FastDctLee (MIT), for the split |
| `hybrid`: 4 long subbands in mixed blocks at 8 kHz | Sieler and Sperschneider, section 4.3.12.1 (sb_amount) |
| `synth`: matrixing, the V vector and the windowed sum | Sieler and Sperschneider, section 4.3.12.6 and Figure 2. dist10, lsf/decoder/decode.c (SubBandSynthesis) |
| `synth`: matrixing through a 32-point DCT, by Byeong Gi Lee's algorithm | B. G. Lee 1984, in the form of Project Nayuki's FastDctLee (MIT). The folding of V onto the DCT follows from the cosine's symmetries |
| `synth`: synthesis window taps | Transcribed from PDMP3, pdmp3.c (g_synth_dtbl), public domain, and rounded to multiples of 2^-16. Every tap agrees with dist10, lsf/decoder/tables/dewindow |
| `layer3`: silent frames where main data is missing, with the filterbanks still running | mpg123, src/libmpg123/layer3.c (III_get_side_info) |
| `layer3`: lower-rate scalefactors, their partitions and preflag | FFmpeg, libavcodec/mpegaudiodec_template.c (lsf_sf_expand) and libavcodec/mpegaudiodec_common.c (ff_lsf_nsf_table). mpg123, src/libmpg123/layer3.c (III_get_scale_factors_2) |
| `layer3`: lower-rate intensity ratios and the intensity scale | FFmpeg, libavcodec/mpegaudiodec_template.c (is_table_lsf). mpg123, src/libmpg123/layer3.c (III_i_stereo) |
| `layer3`: the largest value a lower-rate scalefactor's width holds coding no intensity position | The ISO/IEC 14496-26 references of the MPEG-2 and MPEG-2.5 streams with intensity stereo |
| `layer3`: the highest lower-rate bands taking position 0 when the band below lies under the bound | The ISO/IEC 14496-26 reference of `l3_20245` |
| `mp3`: a last frame that the stream ends inside, decoded from the bytes it holds | FFmpeg, libavcodec/mpegaudiodec_template.c (decode_frame) |
| `layer3`, `hybrid`: the normal window for the lowest subbands of a start or stop block with the mixed flag | FFmpeg, libavcodec/mpegaudiodsp_template.c (ff_imdct36_blocks). mpg123, src/libmpg123/layer3.c (III_hybrid) |
| `synth`, `layer3`: filterbanks started afresh when the number of channels changes | The ISO/IEC 14496-26 references of `l3_10201` |
| `conformance`: decoded samples clipped to the 24-bit range before measuring | The ISO/IEC 14496-26 reference of `l3_10204`, which holds clipped samples |
| `mp3`: the Info frame skipped at the start of a stream | mpg123, src/libmpg123/parse.c (check_lame_tag). FFmpeg, libavformat/mp3dec.c (mp3_parse_vbr_tags) |
| `mp3`: tags between frames stepped over, and a resync past other bytes that empties the bit reservoir | mpg123, src/libmpg123/parse.c (wetwork) |
| `mp3`: the stream ending where frames of another version, layer or sampling frequency start | mpg123, src/libmpg123/parse.c (read_frame with MPG123_NO_FRANKENSTEIN) |
| `mp3`: the length from a tag frame whose byte count matches the audio, otherwise from a walk over the frames | FFmpeg, libavformat/mp3dec.c (mp3_parse_vbr_tags), which keeps the frame count unless the file exceeds the byte count by more than a sixteenth |
| `mp3`: gapless playback, leaving out the encoder delay, the padding and the 529-sample decoder delay | FFmpeg, libavformat/mp3dec.c (mp3_parse_vbr_tags). mpg123, src/libmpg123/frame.c (INT123_frame_gapless_init) and src/libmpg123/frame.h (lastframe). LAME technical FAQ |
| `mp3`: a frame that fails to decode giving silence | mpg123, src/libmpg123/libmpg123.c (decode_the_frame) |
| `mp3`: the frame index, its step doubling as it fills | mpg123, src/libmpg123/index.c (fi_shrink and INT123_fi_add) |
| `frame`, `mp3`: the frames decoded before a seek's target, from the reservoir's reach and the granules the filterbanks carry | The reservoir and filterbank state of this decoder. mpg123, src/libmpg123/frame.c (ignoreframe) decodes a fixed number of frames instead |
| `mp3`: a frame whose CRC does not match decoded all the same | FFmpeg, libavcodec/mpegaudiodec_template.c (handle_crc). mpg123, src/libmpg123/parse.c, which skips the CRC |
| `layer12`: Layer I bit allocation, scalefactors and samples, their order and the joint stereo bound | FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer1). mpg123, src/libmpg123/layer1.c (I_step_one and I_step_two) |
| `layer12`: Layer I requantization | FFmpeg, libavcodec/mpegaudiodec_template.c (l1_unscale). mpg123, src/libmpg123/layer1.c (I_step_two) |
| `layer12`: the scalefactor table, with 0 for index 63 | mpg123, src/libmpg123/layer2.c (INT123_init_layer12_stuff). libmad, layer12.c and sf_table.dat |
| `layer12`: a frame with the allocation 15 failing to decode | mpg123, src/libmpg123/layer1.c (check_balloc). libmad, layer12.c (mad_layer_I) |
| `layer12`: the Layer I CRC covering the bit allocation | libmad, layer12.c (mad_layer_I). The ISO/IEC 14496-26 conformance streams with CRCs |
| `layer12`: Layer II allocation tables and quantization classes | The ISO allocation tables, in runs of subbands that share their classes. Every entry agrees with minimp3, minimp3.h (g_bitalloc_code_tab and L12_subband_alloc_table), mpg123, src/libmpg123/l2tables.h, and libmad, layer12.c (sbquant_table, bitalloc_table, offset_table) and qc_table.dat |
| `layer12`: the Layer II table for each bitrate per channel and sampling frequency | mpg123, src/libmpg123/layer2.c (II_select_table). FFmpeg, libavcodec/mpegaudio.c (ff_mpa_l2_select_table). libmad, layer12.c (mad_layer_II). ISO's bitrates for each table, as TwoLAME's notes on Layer II VBR quote them |
| `layer12`: Layer II allocations, scalefactor selections, scalefactors and granules, their order and the joint stereo bound | FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer2). mpg123, src/libmpg123/layer2.c (II_step_one and II_step_two). libmad, layer12.c (mad_layer_II) |
| `layer12`: Layer II ungrouping and requantization | libmad, layer12.c (II_samples) and qc_table.dat. FFmpeg, libavcodec/mpegaudiodec_template.c (l2_unscale_group) and libavcodec/mpegaudiodec_common.c (ff_division_tabs) |
| `layer12`: a grouped word past three samples' codes giving 0 | mpg123, src/libmpg123/init_layer12.h (INT123_init_layer12) and src/libmpg123/l12tabs.h (grp_3tab, grp_5tab and grp_9tab) |
| `layer12`: the Layer II CRC covering the allocations and selections | libmad, layer12.c (mad_layer_II). FFmpeg, libavcodec/mpegaudiodec_template.c (mp_decode_layer2). The ISO/IEC 14496-26 conformance streams with CRCs |
| `layer12`: the free-format table chosen by the measured bitrate | A free-format stream from TwoLAME 0.4.0, checked against the same tone at a fixed bitrate |
| `frame`, `mp3`: the frames decoded before a Layer I or Layer II seek's target, from the sets of subband samples the synthesis carries | The synthesis filterbank of this decoder |
| `trig`: π and cosines | Machin's formula and the Taylor series of the cosine |

No part adapts another decoder's code. The synthesis window taps are data transcribed from PDMP3.
