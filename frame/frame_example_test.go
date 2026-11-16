// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"fmt"
	"os"
	"slices"

	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// ExampleParse parses the header of an MPEG-1 Layer III frame.
func ExampleParse() {
	head, ok := frame.Parse([]byte{0xff, 0xfb, 0x90, 0x64})
	if !ok {
		return
	}

	fmt.Printf("%d Hz, %d bit/s, %d channels\n", head.SampleRate, head.Bitrate, head.Channels())
	fmt.Printf("%d bytes, %d samples\n", head.Size(), head.Samples())
	// Output:
	// 44100 Hz, 128000 bit/s, 2 channels
	// 417 bytes, 1152 samples
}

// ExampleFind finds the first frame of a stream after bytes that are not audio.
func ExampleFind() {
	frameBytes := slices.Concat([]byte{0xff, 0xfb, 0x90, 0x00}, make([]byte, 413))
	stream := slices.Concat([]byte("junk"), frameBytes, frameBytes)

	at, head, confirmed := frame.Find(stream, nil)

	fmt.Println(at, head.Size(), confirmed)
	// Output: 4 417 true
}

// ExampleHeader_CheckCRC checks the CRC of a protected Layer III frame, which covers its side
// information.
func ExampleHeader_CheckCRC() {
	data, err := os.ReadFile("testdata/protected.mp3")
	if err != nil {
		return
	}

	head, ok := frame.Parse(data)
	if !ok {
		return
	}

	fmt.Println(head.Protected, head.CheckCRC(data[:head.Size()], head.SideInfoSize()*8))
	// Output: true true
}
