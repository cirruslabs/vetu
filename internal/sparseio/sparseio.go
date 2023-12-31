package sparseio

import (
	"bytes"
	"errors"
	"io"
)

const blockSize = 64 * 1024

func Copy(dst io.WriterAt, src io.Reader) error {
	chunk := make([]byte, blockSize)
	zeroedChunk := make([]byte, blockSize)

	var offset int64

	for {
		n, err := src.Read(chunk)

		// Process read bytes (if any) first to correctly
		// handle a case when Read() returns (n > 0, io.EOF)
		if n > 0 {
			// Only write non-zero chunks
			if !bytes.Equal(chunk[:n], zeroedChunk[:n]) {
				if _, err := dst.WriteAt(chunk[:n], offset); err != nil {
					return err
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		offset += int64(n)
	}
}
