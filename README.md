# mp3

A pure-Go MPEG audio decoder for MPEG-1, MPEG-2 and MPEG-2.5 audio in Layers I, II and III.

## Goals

- **Conformance:** full accuracy, as ISO/IEC 11172-4 clause 2.6.3 defines it, on every stream of the ISO/IEC 14496-26 conformance package.
- **Compatibility:** it plays every stream that FFmpeg and mpg123 play.
- **Performance:** fast scalar Go with no allocations per frame.
- **Determinism:** identical output on every architecture.

## Status

The frame header parser is complete. The decoder is in development.

## Packages

| Package | Contents |
| --- | --- |
| [frame](frame) | Frame headers of every layer and version, the confirmed frame search and the CRC check |

## Documentation

- [References](docs/references.md) lists the standards, descriptions, decoders and papers behind the code.
- [Provenance](docs/provenance.md) says what each part was written from.
- [Design](docs/design.md) records the decisions that shape the decoder.

## Licence

AGPL-3.0-or-later.
