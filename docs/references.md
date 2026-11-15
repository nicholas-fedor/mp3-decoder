# References

These are the sources behind the decoder. [Provenance](provenance.md) says which parts each one shaped.

## Standards

| Name | Document | Role |
| --- | --- | --- |
| ISO/IEC 11172-3 | ISO/IEC 11172-3:1993 with Technical Corrigendum 1:1996. <https://www.iso.org/standard/22412.html> | Defines MPEG-1 audio, Layers I, II and III. Paid. Its text is not used |
| ISO/IEC 13818-3 | ISO/IEC 13818-3:1998. <https://www.iso.org/standard/26797.html> | Defines MPEG-2 audio, including the sampling frequencies 16, 22.05 and 24 kHz. Paid and not used, as above |
| ISO/IEC 11172-4 | ISO/IEC 11172-4:1995 with Technical Corrigendum 1:2007. <https://www.iso.org/standard/22691.html> | Free. Clause 2.6.3 defines how a decoder's accuracy is measured |
| ISO/IEC 13818-4 | ISO/IEC 13818-4:2004. <https://www.iso.org/standard/40092.html> | Free. The same procedure for MPEG-2 audio |
| ISO/IEC 14496-26 | ISO/IEC 14496-26:2024, electronic annex. <https://standards.iso.org/iso-iec/14496/-26/ed-2/en/> | Free. The conformance bitstreams and 24-bit reference outputs for MPEG-1, MPEG-2 and MPEG-2.5 audio |
| dist10 | MPEG-2 Audio Simulation Software Distribution 10, ISO MPEG Audio Subgroup Software Simulation Group, 1997. The software of ISO/IEC TR 11172-5 and TR 13818-5. <http://www.mp3-tech.org/programmer/sources/dist10.tgz> | The reference decoder for all three layers |

ISO's licence forbids redistributing the conformance bitstreams, so they are never committed. Tests read them from a cache that a fetch task fills.

## Bitstream Descriptions

| Name | Document | Role |
| --- | --- | --- |
| Sieler and Sperschneider | Martin Sieler and Ralph Sperschneider, "MPEG-Layer3 Bitstream Syntax and Decoding", issue 1.2, 4 March 1997. <https://mp3guessenc.sourceforge.io/MPEG%20Layer3%20Bitstream%20Syntax%20and%20Decoding.pdf> | The Layer III syntax and decoding process for MPEG-1, MPEG-2 and MPEG-2.5, with all Layer III tables |
| LAME info tag | "Mp3 Info Tag rev 1 specifications, draft 0". <http://gabriel.mp3-tech.org/mp3infotag.html> | The Xing and Info header and the LAME extension |
| LAME technical FAQ | <https://lame.sourceforge.io/tech-FAQ.txt> | Encoder and decoder delays |
| ID3v2.3 | <https://id3.org/id3v2.3.0> | ID3v2.3 tag header |
| ID3v2.4 | <https://id3.org/id3v2.4.0-structure> | ID3v2.4 tag header and footer |
| ID3v1 | <https://id3.org/ID3v1> | The 128-byte trailer |
| Lyrics3 v2.00 | <https://id3.org/Lyrics3v2> | The Lyrics3 trailer |
| RFC 3003 | <https://www.rfc-editor.org/rfc/rfc3003.html> | The audio/mpeg media type |
| RFC 5219 | <https://www.rfc-editor.org/rfc/rfc5219.html> | The Layer III frame as header, side information and main data |

## Decoders

| Name | Source | Licence |
| --- | --- | --- |
| FFmpeg | <https://github.com/FFmpeg/FFmpeg>, libavcodec/mpegaudio* | LGPL-2.1-or-later |
| mpg123 | <https://mpg123.org/> | LGPL-2.1 |
| libmad | <https://www.underbit.com/products/mad/> | GPL-2.0-or-later |
| Symphonia | <https://github.com/pdeljanov/Symphonia>, symphonia-bundle-mp3 | MPL-2.0 |
| minimp3 | <https://github.com/lieff/minimp3> | CC0-1.0 |
| PDMP3 | <https://github.com/technosaurus/PDMP3> | Public domain |
| go-mp3 | <https://github.com/hajimehoshi/go-mp3> | Apache-2.0 |

## Implementations of Algorithms

| Name | Source | Licence |
| --- | --- | --- |
| Project Nayuki, fast DCT algorithms | <https://www.nayuki.io/page/fast-discrete-cosine-transform-algorithms>, FastDctLee | MIT |

## Papers

| Name | Document | Role |
| --- | --- | --- |
| B. G. Lee 1984 | Byeong Gi Lee, "A new algorithm to compute the discrete cosine transform", IEEE Transactions on Acoustics, Speech, and Signal Processing 32(6), 1984. doi:10.1109/TASSP.1984.1164443 | Fast DCT for the synthesis filterbank |
| Konstantinides 1994 | K. Konstantinides, "Fast subband filtering in MPEG audio coding", IEEE Signal Processing Letters 1(2), 1994. doi:10.1109/97.300309 | Synthesis matrixing as a DCT |
| S.-W. Lee 2001 | Szu-Wei Lee, "Improved algorithm for efficient computation of the forward and backward MDCT in MPEG audio coder", IEEE Transactions on Circuits and Systems II 48(10), 2001. doi:10.1109/82.974789 | Fast IMDCT |
| Britanak and Rao 2001 | V. Britanak and K. R. Rao, "An efficient implementation of the forward and inverse MDCT in MPEG audio coding", IEEE Signal Processing Letters 8(2), 2001. doi:10.1109/97.895372 | Fast IMDCT |
| Britanak 2011 | V. Britanak, "A survey of efficient MDCT implementations in MP3 audio coding standard", Signal Processing 91(4), 2011. doi:10.1016/j.sigpro.2010.09.009 | Survey of IMDCT algorithms |
