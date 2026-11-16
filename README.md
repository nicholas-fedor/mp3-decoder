<!-- markdownlint-disable MD033 MD041 -->
<div align="center">

<img src=".github/assets/logo.svg" alt="mp3-decoder logo" width="160">

# mp3-decoder

[![Go Reference](https://pkg.go.dev/badge/github.com/nicholas-fedor/mp3-decoder.svg)](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder)
[![Coverage](https://codecov.io/gh/nicholas-fedor/mp3-decoder/graph/badge.svg)](https://codecov.io/gh/nicholas-fedor/mp3-decoder)
[![License](https://img.shields.io/github/license/nicholas-fedor/mp3-decoder)](LICENSE.md)

A pure-Go decoder for MPEG-1, MPEG-2 and MPEG-2.5 audio in Layers I, II and III.

</div>

## Features

- Full-accuracy conformance on ISO's test streams ([Conformance](docs/conformance.md))
- Bit-identical output on every architecture
- Gapless playback using LAME tags
- Sample-exact seeking
- Free-format bitrates
- Recovery from corrupt data, with damage statistics
- Tolerance for out-of-spec streams that mpg123 and FFmpeg accept
- No cgo, assembly or dependencies beyond the standard library
- No allocations per frame
- Decoding at about 890× real time ([Performance](docs/design.md#performance))

## Supported Formats

| Layer | MPEG-1 | MPEG-2 | MPEG-2.5 |
| --- | --- | --- | --- |
| Layer I | Full accuracy | Full accuracy | Decoded |
| Layer II | Full accuracy | Full accuracy | Decoded |
| Layer III | Full accuracy | Full accuracy | Full accuracy |

- Full accuracy means the output matches ISO's reference decodes within the limits ISO/IEC 11172-4 sets.
- Decoded means the format is supported but unmeasured, because ISO publishes no test streams for it.

## Usage

### Prerequisites

- Requires Go 1.27+.

### Installation

To add the module:

```sh
go get github.com/nicholas-fedor/mp3-decoder
```

Add the following import:

```go
import "github.com/nicholas-fedor/mp3-decoder"
```

> [!NOTE]
> The import path ends in `mp3-decoder`, but the package is named `mp3`.

### Decoding a Stream

`mp3.OpenStream` decodes from any `io.Reader`, such as an HTTP response body. `Read` fills the buffer and returns a bare `io.EOF` after the last sample. In this example, `play` stands in for your audio output.

```go
decoder, err := mp3.OpenStream(ctx, response.Body)
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

Each sample is a `[2]float64`:

- Index 0 is the left channel and index 1 the right.
- Mono streams have the same value in both channels.
- Values are not clipped. Full scale is ±1.

The decoder does not close the reader you pass it.

### Decoding a File

`mp3.Open` takes an `io.ReadSeeker`, such as an `*os.File`. It also reports the length in samples and supports seeking.

```go
decoder, err := mp3.Open(ctx, file)
if err != nil {
    return err
}

length, _ := decoder.Len() // Always known for a decoder from mp3.Open.

err = decoder.Seek(ctx, length/2)
if err != nil {
    return err
}
```

### Converting Samples

Many audio APIs expect interleaved 16-bit PCM. This function clips each value to ±1 and scales it to `int16`.

```go
// toInt16 appends samples to pcm as interleaved 16-bit values, clipped to full scale.
func toInt16(pcm []int16, samples [][2]float64) []int16 {
    for _, pair := range samples {
        for _, sample := range pair {
            pcm = append(pcm, int16(math.Round(max(-1, min(sample, 1))*math.MaxInt16)))
        }
    }

    return pcm
}
```

## API

| Method | Does |
| --- | --- |
| `Info` | Returns the sample rate and channel count |
| `Read` | Decodes the next samples into a buffer |
| `Len` | Returns the length in samples, for a decoder from `mp3.Open` |
| `Seek` | Moves to any sample, for a decoder from `mp3.Open` |
| `Position` | Returns the index of the next sample `Read` returns |
| `Stats` | Returns the frame count and counts of the damage the decoder recovered from |

Besides errors from the reader and the context, the package returns these sentinel errors. Check for them with `errors.Is`, since `Seek` wraps `ErrOutOfRange`.

| Error | Returned when |
| --- | --- |
| `ErrNoFrames` | `Open` or `OpenStream` finds no MPEG audio frames |
| `ErrNotSeekable` | `Seek` is called on a decoder from `mp3.OpenStream` |
| `ErrOutOfRange` | `Seek` is given a sample below 0 or above `Len` |

## Packages

| Package | Contents |
| --- | --- |
| [mp3](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder) | The decoder (`Open`, `OpenStream` and `Decoder`) |
| [frame](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder/frame) | Header parsing (`Parse` and `Header`), frame sync (`Find`) and CRC checks |
| [container](https://pkg.go.dev/github.com/nicholas-fedor/mp3-decoder/container) | Tag sizes at the start and end of a stream (`TagSize` and `TrailerSize`), and Xing, Info, LAME and VBRI tags (`ParseInfo`) |

`frame` and `container` don't import the decoder. Use them on their own to read a stream's format, length or tags without decoding any audio.

## Out of Scope

- Encoding audio
- Reading ID3 or APEv2 metadata such as titles (`container` reports only tag sizes)
- Resampling
- Demuxing MP4 or other containers

## Versioning

Releases follow [Semantic Versioning](https://semver.org).

## Documentation

- [References](docs/references.md) lists the standards, papers and decoders the code draws on.
- [Provenance](docs/provenance.md) gives the source of each part of the code.
- [Design](docs/design.md) explains how each part works and why.
- [Conformance](docs/conformance.md) has the results on ISO's test streams.

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) covers setup, testing and the rules for changes.

## License

mp3-decoder is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE.md).
