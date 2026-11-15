// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package conformance measures the decoder against the MPEG audio conformance streams of
// ISO/IEC 14496-26.
//
// ISO publishes the streams and their reference waveforms but forbids redistributing them, so
// they live in a local cache rather than with the code. [Fetch] fills the cache with only the
// files it needs, read from ISO's archives by HTTP range requests and checked against the MD5
// checksums ISO publishes. [Streams] lists the streams, their references, and where each reference
// lies in the decoded output. [Reference.Measure] measures decoded output against a reference as
// ISO/IEC 11172-4 clause 2.6.3 does.
package conformance
