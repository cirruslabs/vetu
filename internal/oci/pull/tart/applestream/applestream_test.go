package applestream_test

import (
	"github.com/cirruslabs/vetu/internal/oci/pull/tart/applestream"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestAppleStream(t *testing.T) {
	testCases := []struct {
		Name     string
		Filename string
	}{
		// Ensure that we correctly processes files with multiple compressed
		// blocks marked as "bv41" (mostly large files). Processing such files
		// requires using the dictionary and updating it after each uncompressed
		// block.
		{
			Name:     "multiple-blocks",
			Filename: "pg2554.txt",
		},
		// Ensure that we correctly process files that contain uncompressed
		// blocks marked as "bv4-" (mostly small files).
		{
			Name:     "uncompressed-blocks",
			Filename: "hello-world.txt",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			// Open the original file
			originalBytes, err := os.ReadFile(filepath.Join("testdata", testCase.Filename))
			require.NoError(t, err)

			// Open the compressed file that was created from the original file
			// using Apple's Compression framework's input filter and decompress it
			compressedFile, err := os.Open(filepath.Join("testdata", testCase.Filename+".lz4"))
			require.NoError(t, err)

			uncompressedBytes, err := io.ReadAll(applestream.NewReader(compressedFile))
			require.NoError(t, err)

			// Ensure that we get the exact same contents
			// as in the original file after decompression
			require.Equal(t, originalBytes, uncompressedBytes)
		})
	}
}
