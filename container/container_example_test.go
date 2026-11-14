// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package container_test

import (
	"fmt"
	"os"
	"slices"

	"github.com/nicholas-fedor/mp3-decoder/container"
	"github.com/nicholas-fedor/mp3-decoder/frame"
)

// ExampleParseInfo reads the Info tag that LAME writes in the first frame of a stream.
func ExampleParseInfo() {
	data, err := os.ReadFile("testdata/cbr.mp3")
	if err != nil {
		return
	}

	head, ok := frame.Parse(data)
	if !ok {
		return
	}

	info, ok := container.ParseInfo(head, data)
	if !ok {
		return
	}

	fmt.Printf("%d frames, delay %d, padding %d\n", info.Frames, info.Delay, info.Padding)
	fmt.Printf("%d samples\n", info.Samples(head))
	// Output:
	// 10 frames, delay 576, padding 944
	// 10000 samples
}

// ExampleID3v2Size finds where the audio starts after an ID3v2 tag.
func ExampleID3v2Size() {
	tag := slices.Concat([]byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 20}, make([]byte, 20))
	stream := slices.Concat(tag, []byte{0xff, 0xfb, 0x90, 0x00})

	fmt.Println(container.ID3v2Size(stream))
	// Output: 30
}

// ExampleTrailerSize finds where the audio ends before an ID3v1 tag.
func ExampleTrailerSize() {
	tag := slices.Concat([]byte("TAG"), make([]byte, 125))
	stream := slices.Concat([]byte{0xff, 0xfb, 0x90, 0x00}, tag)

	fmt.Println(len(stream) - container.TrailerSize(stream))
	// Output: 4
}
