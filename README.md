# mp3

A pure-Go MPEG audio decoder for MPEG-1, MPEG-2 and MPEG-2.5 audio in Layers I, II and III.

## Goals

- **Conformance:** full accuracy, as ISO/IEC 11172-4 clause 2.6.3 defines it, on every stream of the ISO/IEC 14496-26 conformance package.
- **Compatibility:** it plays every stream that FFmpeg and mpg123 play.
- **Performance:** fast scalar Go with no allocations per frame.
- **Determinism:** identical output on every architecture.

## Status

The decoder plays streams of MPEG-1, MPEG-2 and MPEG-2.5 Layer III frames, and decodes every Layer III stream of ISO's conformance package at full accuracy, as [Conformance](docs/conformance.md) records. Layers I and II, seeking and gapless playback are in development.

## Usage

```go
decoder, err := mp3.OpenStream(ctx, file)
if err != nil {
 return err
}

buffer := make([][2]float64, 4096)

for {
 read, err := decoder.Read(buffer)
 play(buffer[:read])

 if errors.Is(err, io.EOF) {
  return nil
 }

 if err != nil {
  return err
 }
}
```

Samples come as left and right pairs, unclipped, with full scale at 1.

## Packages

| Package | Contents |
| --- | --- |
| mp3, this directory | Opening and decoding a stream |
| [frame](frame) | Frame headers of every layer and version, the confirmed frame search and the CRC check |
| [container](container) | ID3v2 headers, end-of-stream tags, and the Xing, Info, LAME and VBRI tags |

## Testing

Each package has white-box unit tests, a black-box integration test that uses only its exported API, examples and fuzz targets. The internal package [mp3test](internal/mp3test) builds bitstreams for tests. It encodes from its own field definitions, so the parsers are not checked against themselves. The [testdata](testdata) directory holds streams that LAME encoded and the samples mpg123 and ffmpeg decode from them, which its `generate.sh` script writes and the tests of several packages read. Its `golden.txt` records the hash of the samples each stream decodes to, which must be the same on every machine. The internal package [conformance](internal/conformance) measures the decoder against ISO's conformance streams, which `task test` fetches into a cache first.

## Documentation

- [References](docs/references.md) lists the standards, descriptions, decoders and papers behind the code.
- [Provenance](docs/provenance.md) says what each part was written from.
- [Design](docs/design.md) records the decisions that shape the decoder.
- [Conformance](docs/conformance.md) records how the decoder measures against ISO's conformance streams.

## Licence

AGPL-3.0-or-later.
