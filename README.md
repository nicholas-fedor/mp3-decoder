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

| Package | What it holds |
| --- | --- |
| [mp3](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder) | `Open`, `OpenStream` and the `Decoder`, which reads, seeks and counts damage |
| [frame](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder/frame) | Frame headers of every layer and version (`Parse` and `Header`), the confirmed frame search (`Find`) and the CRC check |
| [container](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder/container) | The tags at the start of a stream (`TagSize`) and at its end (`TrailerSize`), and the Xing, Info, LAME and VBRI tags (`ParseInfo`) |

`frame` and `container` read a stream's structure without decoding its audio, and neither imports the decoder. A program that needs only a stream's format, length or tags can use them alone. Each package's documentation shows how.

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) explains how to work on the decoder and test it.

## Documentation

- [References](docs/references.md) lists the standards, descriptions, decoders and papers behind the code.
- [Provenance](docs/provenance.md) says what each part was written from.
- [Design](docs/design.md) records the decisions that shape the decoder.
- [Conformance](docs/conformance.md) records how the decoder measures against ISO's conformance streams.

## Licence

AGPL-3.0-or-later.
