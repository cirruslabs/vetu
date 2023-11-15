// Package applestream provides primitives for reading compressed files
// produced by Apple's Compression framework's filters, which we call
// Apple Streams.
//
// No official documentation on Apple Streams exists, except for the public
// reverse-engineering efforts[1].
//
//nolint:lll // work around https://github.com/golangci/golangci-lint/issues/3983
// [1]: https://github.com/libyal/dtformats/blob/main/documentation/Apple%20Unified%20Logging%20and%20Activity%20Tracing%20formats.asciidoc#26-lz4-compressed-data

package applestream

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/pierrec/lz4/v4"
	"io"
)

const maxAllocatedBytesPerBlockMarker = 128 * 1024 * 1024

var (
	ErrFailed = errors.New("decompression of Apple Stream failed")

	blockMarkerCompressed   = []byte("bv41")
	blockMarkerUncompressed = []byte("bv4-")
	blockMarkerEOF          = []byte("bv4$")
)

type Reader struct {
	underlyingReader    io.Reader
	underlyingReaderEOF bool
	lz4Dict             []byte
	uncompressed        bytes.Buffer
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		underlyingReader: r,
	}
}

func (reader *Reader) Read(p []byte) (n int, err error) {
	// Do not process next blocks if we have enough uncompressed data
	// to feed to the reader. This prevents high memory consumption.
	if reader.uncompressed.Len() >= len(p) {
		return reader.uncompressed.Read(p)
	}

	if !reader.underlyingReaderEOF {
		if err := reader.processNextBlock(); err != nil {
			return 0, err
		}
	}

	return reader.uncompressed.Read(p)
}

func (reader *Reader) processNextBlock() error {
	// Read block marker
	blockMarker := make([]byte, 4)
	if _, err := io.ReadFull(reader.underlyingReader, blockMarker); err != nil {
		return fmt.Errorf("%w: failed to read block marker: %v", ErrFailed, err)
	}

	switch {
	case bytes.Equal(blockMarker, blockMarkerCompressed):
		// Read compressed block's header
		header := make([]byte, 8)

		if _, err := io.ReadFull(reader.underlyingReader, header); err != nil {
			return fmt.Errorf("%w: failed to read the header of a compressed block: %v",
				ErrFailed, err)
		}

		// Parse compressed block's header
		uncompressedSize := binary.LittleEndian.Uint32(header[:4])
		compressedSize := binary.LittleEndian.Uint32(header[4:])

		// DoS check
		if (uncompressedSize + compressedSize) > maxAllocatedBytesPerBlockMarker {
			return fmt.Errorf("%w: block's uncompressed and compressed sizes "+
				"(%d and %d bytes respectively) exceed the per-block limit of %d bytes",
				ErrFailed, uncompressedSize, compressedSize, maxAllocatedBytesPerBlockMarker)
		}

		// Reader compressed block's data
		compressedBytes := make([]byte, compressedSize)
		if _, err := io.ReadFull(reader.underlyingReader, compressedBytes); err != nil {
			return fmt.Errorf("%w: failed to read compressed bytes: %v", ErrFailed, err)
		}

		// Decompress compressed block's data
		uncompressedBytes := make([]byte, uncompressedSize)
		uncompressedN, err := lz4.UncompressBlockWithDict(compressedBytes, uncompressedBytes, reader.lz4Dict)
		if err != nil {
			return fmt.Errorf("%w: LZ4 failed to decompress the block: %v", ErrFailed, err)
		}

		// Update LZ4 dictionary
		reader.lz4Dict = uncompressedBytes[:uncompressedN]

		// Store the uncompressed data
		_, _ = reader.uncompressed.Write(uncompressedBytes[:uncompressedN])
	case bytes.Equal(blockMarker, blockMarkerUncompressed):
		// Read uncompressed block's header
		header := make([]byte, 4)

		if _, err := io.ReadFull(reader.underlyingReader, header); err != nil {
			return fmt.Errorf("%w: failed to read the header of an uncompressed block: %v",
				ErrFailed, err)
		}

		// Parse uncompressed block's header
		uncompressedSize := binary.LittleEndian.Uint32(header[:4])

		// DoS check
		if uncompressedSize > maxAllocatedBytesPerBlockMarker {
			return fmt.Errorf("%w: block's uncompressed size (%d bytes) exceeds "+
				"the per-block limit of %d bytes", ErrFailed, uncompressedSize,
				maxAllocatedBytesPerBlockMarker)
		}

		// Read uncompressed block's data
		uncompressedBytes := make([]byte, uncompressedSize)

		if _, err := io.ReadFull(reader.underlyingReader, uncompressedBytes); err != nil {
			return fmt.Errorf("%w: failed to read uncompressed bytes: %v", ErrFailed, err)
		}

		// Update LZ4 dictionary
		reader.lz4Dict = uncompressedBytes

		// Store the uncompressed data
		_, _ = reader.uncompressed.Write(uncompressedBytes)
	case bytes.Equal(blockMarker, blockMarkerEOF):
		reader.underlyingReaderEOF = true
	default:
		return fmt.Errorf("%w: encountered an unsupported block marker during read: %s",
			ErrFailed, hex.EncodeToString(blockMarker))
	}

	return nil
}
