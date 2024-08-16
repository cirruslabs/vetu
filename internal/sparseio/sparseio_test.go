package sparseio_test

import (
	"github.com/cirruslabs/vetu/internal/sparseio"
	"github.com/dustin/go-humanize"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestCopySmall(t *testing.T) {
	// Create a small file
	originalFilePath := filepath.Join(t.TempDir(), "original.txt")
	err := os.WriteFile(originalFilePath, []byte("Hello, World!\n"), 0600)
	require.NoError(t, err)

	// Sparsely copy it
	sparseFilePath := filepath.Join(t.TempDir(), "sparse.txt")
	copySparse(t, originalFilePath, sparseFilePath)

	// Ensure that both files have identical contents
	require.Equal(t, fileDigest(t, originalFilePath), fileDigest(t, sparseFilePath))
}

//nolint:gosec // we don't need cryptographically secure randomness here
func TestCopyRandomized(t *testing.T) {
	// Create a sufficiently large file that contains
	// interleaved random-filled and zero-filled parts
	originalFilePath := filepath.Join(t.TempDir(), "original.bin")
	originalFile, err := os.Create(originalFilePath)
	require.NoError(t, err)

	var wroteBytes int64

	for wroteBytes < 1*humanize.GByte {
		chunk := randomlySizedChunk(1*humanize.KByte, 4*humanize.MByte)

		// Randomize the contents of some chunks
		if rand.Intn(2) == 1 {
			_, err = rand.Read(chunk)
			require.NoError(t, err)
		}

		n, err := originalFile.Write(chunk)
		require.NoError(t, err)

		wroteBytes += int64(n)
	}

	require.NoError(t, originalFile.Close())

	// Sparsely copy the original file
	sparseFilePath := filepath.Join(t.TempDir(), "sparse.bin")
	copySparse(t, originalFilePath, sparseFilePath)

	// Ensure that the copied file has the same contents as the original file
	require.Equal(t, fileDigest(t, originalFilePath), fileDigest(t, sparseFilePath))
}

func copySparse(t *testing.T, originalFilePath string, sparseFilePath string) {
	originalFile, err := os.Open(originalFilePath)
	require.NoError(t, err)

	originalFileInfo, err := originalFile.Stat()
	require.NoError(t, err)

	sparseFile, err := os.Create(sparseFilePath)
	require.NoError(t, err)

	require.NoError(t, sparseFile.Truncate(originalFileInfo.Size()))
	require.NoError(t, sparseio.Copy(sparseFile, originalFile))

	require.NoError(t, originalFile.Close())
	require.NoError(t, sparseFile.Close())
}

//nolint:gosec // we don't need cryptographically secure randomness here
func randomlySizedChunk(minBytes int, maxBytes int) []byte {
	return make([]byte, rand.Intn(maxBytes-minBytes+1)+minBytes)
}

func fileDigest(t *testing.T, path string) digest.Digest {
	file, err := os.Open(path)
	require.NoError(t, err)

	digest, err := digest.FromReader(file)
	require.NoError(t, err)

	require.NoError(t, file.Close())

	return digest
}
