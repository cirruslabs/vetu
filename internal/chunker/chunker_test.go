package chunker_test

import (
	"bytes"
	cryptorand "crypto/rand"
	chunkerpkg "github.com/cirruslabs/vetu/internal/chunker"
	"github.com/opencontainers/go-digest"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestSimple(t *testing.T) {
	const chunkSize = 1 * 1024 * 1024

	var expectedChunks []*chunkerpkg.Chunk

	for i := 0; i < 10; i++ {
		data, err := io.ReadAll(io.LimitReader(cryptorand.Reader, chunkSize))
		require.NoError(t, err)

		expectedChunks = append(expectedChunks, &chunkerpkg.Chunk{
			Data:               data,
			UncompressedSize:   chunkSize,
			UncompressedDigest: digest.FromBytes(data),
		})
	}

	chunker := chunkerpkg.NewChunker(chunkSize, func(w io.Writer) (io.WriteCloser, error) {
		return WriteNopCloser(w), nil
	})

	go func() {
		defer chunker.Close()

		expectedReader := io.MultiReader(lo.Map(expectedChunks, func(chunk *chunkerpkg.Chunk, index int) io.Reader {
			return bytes.NewReader(chunk.Data)
		})...)

		_, err := io.Copy(chunker, expectedReader)
		require.NoError(t, err)
	}()

	var actualChunks []*chunkerpkg.Chunk

	for chunk := range chunker.Chunks() {
		actualChunks = append(actualChunks, chunk)
	}

	require.Equal(t, expectedChunks, actualChunks)
}

type writeNopCloser struct {
	io.Writer
}

func WriteNopCloser(w io.Writer) io.WriteCloser {
	return &writeNopCloser{w}
}

func (nopCloser *writeNopCloser) Close() error {
	return nil
}
