# Contributing to mp3-decoder

Thank you for helping with the decoder. This guide explains how to work on it and test it. Coding agents also follow [AGENTS.md](AGENTS.md).

## Contents

- [Contents](#contents)
- [Setup](#setup)
- [The Gate](#the-gate)
- [Other Tasks](#other-tasks)
- [Packages](#packages)
- [Decoder Rules](#decoder-rules)
  - [Dependencies](#dependencies)
  - [Determinism](#determinism)
  - [Tables](#tables)
  - [Damage](#damage)
  - [Compatibility](#compatibility)
  - [Performance](#performance)
  - [Sources](#sources)
- [Code Style](#code-style)
- [Tests](#tests)
  - [Fixtures](#fixtures)
  - [Conformance](#conformance)
- [Documentation](#documentation)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Releases](#releases)
- [AI-Assisted Contributions](#ai-assisted-contributions)
- [License](#license)

## Setup

| Tool | Needed for |
| --- | --- |
| [Go](https://go.dev/dl/), the version in `go.mod` | Building and testing |
| [Task](https://taskfile.dev) | Running the tasks |
| [golangci-lint](https://golangci-lint.run) v2 | `task fmt` and `task lint` |
| [betteralign](https://github.com/dkorunic/betteralign) | Aligning struct fields in `task fmt` and `task lint` |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | `task vuln` |
| qemu-aarch64, optional | Running `task test:fma` on arm64 |
| ffmpeg, lame, twolame and mpg123, optional | Regenerating the fixtures |

```sh
git clone https://github.com/nicholas-fedor/mp3-decoder.git
cd mp3-decoder
task check
```

The first `task check` downloads about 43 MB of ISO's conformance streams. The race detector needs cgo and a C toolchain. Where there is none, set `MP3_DECODER_RACE` to an empty value.

## The Gate

Every change must pass `task check`. It runs these tasks in order:

1. `fmt` formats the code with golangci-lint and aligns struct fields with betteralign.
2. `vet` runs go vet for the host, then for linux/arm64, darwin/arm64 and windows/amd64.
3. `modernize` fails when `go fix` suggests a change. `FIX=true task modernize` applies them.
4. `test` fills the conformance cache, then runs every test with the race detector and fails below 80% coverage.
5. `test:fma` reruns the tests at `GOAMD64=v3`, and on arm64 under qemu-aarch64 where it is installed.
6. `lint` checks that `go.mod` is tidy, then runs golangci-lint and betteralign. It must report 0 issues.
7. `vuln` reports known vulnerabilities with govulncheck.

CI runs the tests on Linux, macOS and Windows. It also runs the linters, gosec and govulncheck. It does not run the conformance test or `test:fma`, so only `task check` runs the whole gate.

## Other Tasks

| Task | What it does |
| --- | --- |
| `task conformance` | Fills the conformance cache, fetching only the files it lacks |
| `task bench` | Runs the benchmarks with allocation counts |
| `task fuzz` | Runs each fuzz target for 30 seconds. `FUZZTIME=10m` sets another length |
| `task update` | Updates every dependency to its latest release and tidies `go.mod` |
| `task clean` | Removes the coverage profile |

## Packages

The public packages are `mp3` at the root, [frame](frame) and [container](container). The [README](README.md#packages) describes them. The internal packages hold the decoding stages and the test tools.

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

## Decoder Rules

These rules keep the decoder's goals. [Design](docs/design.md) gives the reasons for each.

### Dependencies

Production code imports only the standard library and the module's own packages. Tests may also import testify. The leaf packages `frame` and `container` import nothing of the module but `frame`, so a program that reads headers or tags links no decoder. depguard enforces both rules.

### Determinism

The decoder gives the same bits on every architecture. Go may fuse a product with a later sum into one multiply-add, which rounds once instead of twice. Only a conversion forbids it. Every floating-point product is therefore written as `float64(a * b)`, and a float is never scaled with `*=`.

- `TestProductsRounded` reports every product that is not converted where it is made.
- `TestGolden` compares a hash of each fixture's decoded samples with `testdata/golden.txt`.
- `task test:fma` runs both where products fuse.

A change that alters the output on purpose records the new hashes with `go test -run '^TestGolden$' -update .` and says why the output changed.

### Tables

Normative tables are literals in the code. A table derived from a closed form is computed exactly with `math/big` by a test of its package, which writes it into a `*_gen.go` file. `go generate ./...` rewrites them all, and the same tests fail when a generated file is stale. Never edit a `*_gen.go` file by hand. Nothing is computed when a program starts.

### Damage

Damage in a bitstream never ends a stream. The decoder turns it into silence or a resync, and `Stats` counts it. `Read` returns an error only when the source fails, and an unwrapped `io.EOF` after the last sample.

### Compatibility

Where a written description is stricter than mpg123 and FFmpeg, the decoder accepts what they accept. Reserved and forbidden header values stay rejected, since the frame search depends on them.

### Performance

The decoder is scalar Go with no assembly. `Read` and `Seek` allocate nothing, and an integration test fails on any allocation. A change on the decoding path reports `task bench` before and after.

### Sources

Code comments cite no sources. [References](docs/references.md) lists every source, and [Provenance](docs/provenance.md) says what each part was written from. Code adapted from another decoder gets a row in Provenance with its licence. Any source whose licence combines with AGPL-3.0 may be studied and adapted.

## Code Style

- Every Go file starts with the copyright line and `SPDX-License-Identifier: AGPL-3.0-or-later`.
- Every package has a `doc.go`. Every declaration has a doc comment, with `Parameters:` and `Returns:` sections on functions and methods.
- A doc comment opens with a one-sentence summary. Details go in a paragraph of their own.
- Comments describe what the code does now and why, never its history.
- Names follow the bitstream, such as `BigValue` and `Count1`. A type is named for the data it holds, never for an operation or a feature.
- Errors are package sentinels such as `ErrOutOfRange`, wrapped with `%w` and checked with `errors.Is`.
- Struct literals set every field, and every `switch` has a `default`.
- Lines stay within 120 columns, and files within 400 lines.
- Imports form three groups: the standard library, third-party modules, then `github.com/nicholas-fedor/mp3-decoder`.
- Every `//nolint` names the linter and gives a reason.

## Tests

Each production file has a white-box test file. A package that decodes or parses also has the other files below.

| File | Holds |
| --- | --- |
| `<file>_test.go` | White-box tests of one production file |
| `<pkg>_integration_test.go` | Black-box tests that use only the package's exported API |
| `<pkg>_fuzz_test.go` | Fuzz targets |
| `<pkg>_benchmark_test.go` | Benchmarks of the stages on the decoding path |
| `<pkg>_example_test.go` | Examples of a public package's API |

- **Style:** tests are table-driven with `t.Run` and call `t.Parallel()` wherever it is safe. Every test, table and helper has a doc comment. testify's `require` makes fatal checks and `assert` the rest.
- **Bitstreams:** the internal package [mp3test](internal/mp3test) builds bitstreams for tests, from frame headers and tags to the audio data of each layer. It encodes from its own field definitions, so the parsers are not checked against themselves.
- **Fuzzing:** a crasher stays as a regression seed under the package's `testdata/fuzz/<FuzzName>/`.
- **Network:** `go test` never reaches the network. Only the fetch command does.
- **Bugs:** a test that exposes a bug stays in the suite. Fix the bug rather than weaken the test, its tolerance or a golden hash.

### Fixtures

The [testdata](testdata) directory holds real streams and what other decoders make of them. The tests of several packages read them.

| Files | Contents |
| --- | --- |
| `*.mp3` | Layer III streams that LAME encoded |
| `*.mp2` | Layer II streams that TwoLAME encoded |
| `*.mpg123.f32`, `*.ffmpeg.f32` | The samples mpg123 and ffmpeg decode from the streams, as references |
| `golden.txt` | The hash of the samples each stream decodes to |
| `generate.sh` | The script that writes the streams and references |
| `versions.txt` | The versions of the tools that wrote them |

After `testdata/generate.sh` rewrites the fixtures, record their new hashes with `go test -run '^TestGolden$' -update .`

### Conformance

The internal package [conformance](internal/conformance) measures the decoder against the streams of the ISO/IEC 14496-26 conformance package. ISO's licence forbids passing the streams on, so they are never committed. `task conformance` fetches them into a cache, by default `$XDG_CACHE_HOME/mp3-decoder/conformance`, and checks them against ISO's MD5 manifest.

- `MP3_DECODER_CONFORMANCE` names the cache. The tasks set it, and the conformance test reads it.
- Without it, `go test` skips the conformance test. With it, a missing stream fails the test.
- `go test -run '^TestConformance$' -update ./internal/conformance` rewrites the results and the census in [Conformance](docs/conformance.md).

## Documentation

- [README.md](README.md) is for users of the module. It shows what the decoder does and how to call it.
- [Design](docs/design.md) records the decisions behind each part. A change of behavior updates its section and the contents list in the same commit.
- [References](docs/references.md) and [Provenance](docs/provenance.md) gain a row for each new source.
- [Conformance](docs/conformance.md) is written by its test. Never edit its results by hand.
- Markdown is never hard-wrapped, and headings capitalize every major word.

## Commit Messages

The project uses [Conventional Commits](https://www.conventionalcommits.org) with the Angular types:

```text
<type>(<scope>): <subject>

- <bullet point 1>
- <bullet point 2>
- <bullet point 3>
```

- **Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore` and `revert`.
- **Scope** names the package the change lives in, such as `frame` or `layer3`. A change to the root package, or one across packages, has no scope.
- **Subject:** imperative, lower case, no trailing period.
- **Body:** up to 5 bullet points, most significant first, each imperative and starting with a capital letter. No prose body.
- Add a footer only for a breaking change or an issue reference.

```text
fix(container): read Xing, Info and VBRI tags only in Layer III frames

- Decode the first frame of a Layer I or Layer II stream as audio whatever its bytes hold, as mpg123 and FFmpeg do
- Keep the Layer III decoder delay from trimming Layer I and Layer II streams
```

## Pull Requests

- Open pull requests against `main`. Keep each one to a single change.
- The title uses the commit format, such as `perf(hybrid): skip the IMDCT of zero subbands`.
- The body follows this layout:

  ```markdown
  This PR <brief 2-3 sentence description>

  ## Problem

  <prose>

  ## Solution

  <prose>

  ## Changes

  - <flat bullet>
  - <flat bullet>
  ```

- State in the body that `task check` passes, since CI does not run the conformance test. Every CI check must pass too.

## Releases

Maintainers release from an up-to-date `main` with `task tag:bump-patch`, `task tag:bump-minor` or `task tag:bump-major`. Each finds the latest `vX.Y.Z` tag on origin and asks before it tags the next version. Then it pushes the tag. It refuses to tag in these cases:

- The branch is not `main`.
- The tree has changes.
- `main` differs from origin's.
- The commit already has a release tag.

The Release workflow checks the tag and runs the tests. Then it asks the Go module proxy and pkg.go.dev for the new version. On every push to `main`, the Update Changelog workflow opens a pull request with the changelog that git-cliff writes from the commits.

## AI-Assisted Contributions

AI tools are welcome, on the same terms as any other code:

- You review and understand every line you submit.
- You can explain what it does and why.
- It passes the gate and the review like any other change.

## License

By contributing, you agree that your contributions are licensed under the [GNU Affero General Public License v3.0 or later](LICENSE.md).
