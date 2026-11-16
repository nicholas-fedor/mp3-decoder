// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

// Package bits reads MPEG audio bitstreams bit by bit.
//
// A [Reader] reads up to 32 bits at a time, most significant bit first. Each read loads the 64 bits
// from its position on in one step. Reads past the end of its bytes give zeros, so the decoders
// bound their reads by position rather than checking each one.
package bits
