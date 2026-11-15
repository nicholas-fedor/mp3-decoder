# mp3

A pure-Go MPEG audio decoder for MPEG-1, MPEG-2 and MPEG-2.5 audio in Layers I, II and III.

## Goals

- **Conformance:** full accuracy, as ISO/IEC 11172-4 clause 2.6.3 defines it, on every stream of the ISO/IEC 14496-26 conformance package.
- **Compatibility:** it plays every stream that FFmpeg and mpg123 play.
- **Performance:** fast scalar Go with no allocations per frame.
- **Determinism:** identical output on every architecture.

## Status

The decoder decodes every stream of ISO's conformance package at full accuracy, as [Conformance](docs/conformance.md) records.

| Layer | MPEG-1 | MPEG-2 | MPEG-2.5 |
| --- | --- | --- | --- |
| Layer I | Full accuracy | Full accuracy | Decoded as MPEG-2 is, with no conformance streams to measure |
| Layer II | Full accuracy | Full accuracy | Decoded as MPEG-2 is, with no conformance streams to measure |
| Layer III | Full accuracy | Full accuracy | Full accuracy |

It also does the following:

- **Gapless playback:** it leaves out the encoder delay and padding a LAME tag gives.
- **Seeking:** it seeks to any sample exactly, to the same samples a decode from the start gives.
- **Free format:** it decodes free-format streams, which FFmpeg refuses.
- **Damage:** it carries on past damage and junk, and counts what it met.

## Usage

### Adding the Module

```sh
go get github.com/nicholas-fedor/mp3-decoder
```

The package at the module's root is named `mp3`, so an import gives it that name.

```go
import "github.com/nicholas-fedor/mp3-decoder"
```

### Decoding a Stream

`mp3.OpenStream` takes any `io.Reader`, such as a network stream. `Read` fills a buffer with samples and returns `io.EOF`, unwrapped, after the last one.

```go
decoder, err := mp3.OpenStream(ctx, stream)
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

Samples come as left and right pairs, unclipped, with full scale at 1. A mono stream gives each sample to both channels. The decoder does not close its source.

### Decoding a File

`mp3.Open` takes an `io.ReadSeeker`. It also finds the number of samples the file decodes to, and it can seek.

```go
decoder, err := mp3.Open(ctx, file)
if err != nil {
    return err
}

length, _ := decoder.Len()

err = decoder.Seek(ctx, length/2)
if err != nil {
    return err
}
```

### Methods

| Method | What it gives |
| --- | --- |
| `Info` | The sampling frequency and the number of channels |
| `Read` | The next samples |
| `Len` | The number of samples, for a decoder from `mp3.Open` |
| `Seek` | A move to any sample, for a decoder from `mp3.Open` |
| `Position` | The number of the sample `Read` gives next |
| `Stats` | Counts of the damage decoding carried on past |

## Packages

### Public Packages

| Package | What it holds |
| --- | --- |
| mp3, this directory | `Open`, `OpenStream` and the `Decoder`, which reads, seeks and counts damage |
| [frame](frame) | Frame headers of every layer and version (`Parse` and `Header`), the confirmed frame search (`Find`) and the CRC check |
| [container](container) | The tags at the start of a stream (`TagSize`) and at its end (`TrailerSize`), and the Xing, Info, LAME and VBRI tags (`ParseInfo`) |

### Using frame and container Alone

The leaf packages read a stream's structure without decoding its audio. A program that needs only a stream's format, length or tags can use them alone:

- **Probing a stream:** `frame.Find` finds the first frame that the header after it confirms, and its `Header` gives the version, layer, sampling frequency, bitrate and channels.
- **Length and gapless padding:** `container.ParseInfo` reads a tag frame, and `Info.Samples` gives the number of samples the stream encodes, less the encoder delay and padding when the tag gives them.
- **Seeking by bytes:** `Info.OffsetAt` turns a share of the stream into a byte offset through the tag's seek table, and `Header.Preroll` says how many frames before it a decoder must start to decode exactly.
- **Finding the audio:** `container.TagSize` and `container.TrailerSize` give the sizes of the tags before and after it.

### Internal Packages

| Package | Role |
| --- | --- |
| [internal/bits](internal/bits) | Reads bitstreams bit by bit |
| [internal/huffman](internal/huffman) | Decodes the Huffman-coded spectral values of a Layer III granule, with the code tables in `internal/huffman/codebook` |
| [internal/layer3](internal/layer3) | Decodes the audio data of Layer III frames, as [Layer III Decoding](docs/design.md#layer-iii-decoding) describes |
| [internal/hybrid](internal/hybrid) | Turns the spectral lines of a Layer III granule into polyphase subband samples |
| [internal/layer12](internal/layer12) | Decodes the audio data of Layer I and Layer II frames, as [Layer I Decoding](docs/design.md#layer-i-decoding) and [Layer II Decoding](docs/design.md#layer-ii-decoding) describe |
| [internal/synth](internal/synth) | Turns polyphase subband samples into PCM samples, for every layer |
| [internal/trig](internal/trig) | Computes the exact sines and cosines of the generated tables, for their tests |
| [internal/mp3test](internal/mp3test) | Builds bitstreams for tests |
| [internal/conformance](internal/conformance) | Measures the decoder against ISO's conformance streams, with the command that fetches them in `internal/conformance/fetch` |

## Testing

Every package has white-box unit tests. The packages that decode or parse also have these:

- A black-box integration test that uses only the package's exported API.
- Fuzz targets.
- Benchmarks for the stages on the decoding path.

The public packages, mp3, [frame](frame) and [container](container), also have examples.

### Test Bitstreams

The internal package [mp3test](internal/mp3test) builds bitstreams for tests, from frame headers and tags to the audio data of each layer. It encodes from its own field definitions, so the parsers are not checked against themselves.

### Fixtures

The [testdata](testdata) directory holds real streams and what other decoders make of them. The tests of several packages read them.

| Files | Contents |
| --- | --- |
| `*.mp3` | Layer III streams that LAME encoded |
| `*.mp2` | Layer II streams that TwoLAME encoded |
| `*.mpg123.f32`, `*.ffmpeg.f32` | The samples mpg123 and ffmpeg decode from the streams, as references |
| `golden.txt` | The hash of the samples each stream decodes to, which must be the same on every machine |
| `generate.sh` | The script that writes the streams and references |
| `versions.txt` | The versions of the tools that wrote them |

### Conformance

The internal package [conformance](internal/conformance) measures the decoder against ISO's conformance streams. ISO's licence forbids passing the streams on, so `task test` fetches them into a cache first. [Conformance](docs/conformance.md) records the results.

## Documentation

- [References](docs/references.md) lists the standards, descriptions, decoders and papers behind the code.
- [Provenance](docs/provenance.md) says what each part was written from.
- [Design](docs/design.md) records the decisions that shape the decoder.
- [Conformance](docs/conformance.md) records how the decoder measures against ISO's conformance streams.

## Licence

AGPL-3.0-or-later.
