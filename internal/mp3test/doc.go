// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

// Package mp3test builds MPEG audio bitstreams for tests.
//
// It builds frame headers, tags and trailers, and the audio data of each layer down to its side
// information, scalefactors and Huffman codes. It encodes from its own field definitions, so the
// parsers and decoders are not checked against themselves.
package mp3test
