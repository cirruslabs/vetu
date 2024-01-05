package temporary_test

import (
	cryptorand "crypto/rand"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/dustin/go-humanize"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicallyCopyThrough(t *testing.T) {
	t.Setenv("VETU_HOME", filepath.Join(t.TempDir(), ".vetu"))

	tmpDir := t.TempDir()

	// Create a source directory
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.Mkdir(srcDir, 0700))

	// Add a small-sized text file to it
	err := os.WriteFile(filepath.Join(srcDir, "text.txt"), []byte("Hello, World!\n"), 0600)
	require.NoError(t, err)

	// Add a medium-sized binary file to it
	buf := make([]byte, 64*humanize.MByte)
	_, err = cryptorand.Read(buf)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "binary.bin"), buf, 0600)
	require.NoError(t, err)

	// Copy source directory contents to destination directory
	dstVMDir, err := temporary.CreateFrom(srcDir)
	require.NoError(t, err)

	// Ensure that the files copied are identical
	// to the ones in the source directory
	require.Equal(t, fileDigest(t, filepath.Join(dstVMDir.Path(), "text.txt")), digest.FromString("Hello, World!\n"))
	require.Equal(t, fileDigest(t, filepath.Join(dstVMDir.Path(), "binary.bin")), digest.FromBytes(buf))
}

func fileDigest(t *testing.T, path string) digest.Digest {
	file, err := os.Open(path)
	require.NoError(t, err)

	digest, err := digest.FromReader(file)
	require.NoError(t, err)

	require.NoError(t, file.Close())

	return digest
}
