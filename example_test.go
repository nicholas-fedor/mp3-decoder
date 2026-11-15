// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/nicholas-fedor/mp3-decoder"
)

// ExampleOpenStream decodes a stream to its end and counts its samples, which are the 0.5 s at
// 44.1 kHz its encoder took in.
func ExampleOpenStream() {
	file, err := os.Open("testdata/mpeg1-joint.mp3")
	if err != nil {
		fmt.Println(err)

		return
	}

	defer file.Close()

	decoder, err := mp3.OpenStream(context.Background(), file)
	if err != nil {
		fmt.Println(err)

		return
	}

	samples := 0
	buffer := make([][2]float64, 4096)

	for {
		read, err := decoder.Read(buffer)
		samples += read

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			fmt.Println(err)

			return
		}
	}

	fmt.Println(samples, "samples")
	// Output: 22050 samples
}
