// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// remote reads a file over HTTP by range requests, a block at a time, keeping the last block.
//
// It serves one reader at a time.
type remote struct {
	//nolint:containedctx // io.ReaderAt takes no context, and a remote lives for one Fetch call.
	ctx context.Context
	// client sends the requests.
	client *http.Client
	// url is the file's URL.
	url string
	// block holds the bytes of the last block read.
	block []byte
	// at is the offset of block in the file.
	at int64
	// size is the file's size.
	size int64
}

// blockSize is the most bytes one range request asks for.
const blockSize = 1 << 20

// ErrNoRange is returned when a server answers a range request with anything but the range.
var ErrNoRange = errors.New("conformance: server does not answer range requests")

// openRemote opens a file over HTTP, reading its first block to learn its size.
//
// Parameters:
//   - ctx: cancels every request the remote sends.
//   - client: sends the requests.
//   - url: the file's URL.
//
// Returns:
//   - *remote: the file.
//   - error: [ErrNoRange] or a request error.
func openRemote(ctx context.Context, client *http.Client, url string) (*remote, error) {
	file := &remote{ctx: ctx, client: client, url: url, block: make([]byte, 0, blockSize), at: 0, size: 0}

	err := file.fetch(0)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// ReadAt reads len(dst) bytes from the file at an offset.
//
// Parameters:
//   - dst: where the bytes go.
//   - off: the offset in the file.
//
// Returns:
//   - int: the number of bytes read.
//   - error: [io.EOF] when the file ends first, or a request error.
func (r *remote) ReadAt(dst []byte, off int64) (int, error) {
	read := 0

	for read < len(dst) {
		at := off + int64(read)
		if at >= r.size {
			return read, io.EOF
		}

		if at < r.at || at >= r.at+int64(len(r.block)) {
			err := r.fetch(at - at%blockSize)
			if err != nil {
				return read, err
			}
		}

		read += copy(dst[read:], r.block[at-r.at:])
	}

	return read, nil
}

// fetch reads the block at an offset into block, and learns the file's size from the answer.
//
// Parameters:
//   - at: the block's offset, a multiple of the block size.
//
// Returns:
//   - error: [ErrNoRange] or a request error.
func (r *remote) fetch(at int64) error {
	request, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.url, http.NoBody)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", r.url, err)
	}

	request.Header.Set("Range", "bytes="+strconv.FormatInt(at, 10)+"-"+strconv.FormatInt(at+blockSize-1, 10))

	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", r.url, err)
	}

	defer response.Body.Close()

	var first, last, size int64

	_, err = fmt.Sscanf(response.Header.Get("Content-Range"), "bytes %d-%d/%d", &first, &last, &size)
	if response.StatusCode != http.StatusPartialContent || err != nil || first != at || last < first {
		return fmt.Errorf("%w: %s answers %q with %s", ErrNoRange, r.url, request.Header.Get("Range"),
			response.Status)
	}

	r.block = r.block[:last-first+1]

	_, err = io.ReadFull(response.Body, r.block)
	if err != nil {
		r.block = r.block[:0]

		return fmt.Errorf("fetch %s: %w", r.url, err)
	}

	r.at, r.size = at, size

	return nil
}
