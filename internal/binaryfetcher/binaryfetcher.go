package binaryfetcher

import (
	"context"
	"github.com/cirruslabs/vetu/internal/homedir"
	"io"
	"os"
	"path/filepath"
)

type FetchFunc func(ctx context.Context, binaryFile io.Writer) error

func GetOrFetch(ctx context.Context, fetchFunc FetchFunc, binaryName string, executable bool) (string, error) {
	// Determine the binary path
	binaryPath, err := binaryPath(binaryName)
	if err != nil {
		return "", err
	}

	// Use the cached binary if possible
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Create a temporary directory on the same filesystem
	// to avoid rename(2) failing with EXDEV errno due to
	// "/tmp" being mounted as tmpfs on some systems
	homeDir, err := homedir.Path()
	if err != nil {
		return "", err
	}

	tmpDir := filepath.Join(homeDir, "tmp")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	// Run the user-provided function to fetch the binary file
	// if not available in the cache
	binaryFile, err := os.CreateTemp(tmpDir, "vetu-binary-file-*")
	if err != nil {
		return "", err
	}

	if err := fetchFunc(ctx, binaryFile); err != nil {
		_ = binaryFile.Close()
		_ = os.Remove(binaryFile.Name())

		return "", err
	}

	// Make the binary executable if requested
	if executable {
		if err := binaryFile.Chmod(0755); err != nil {
			_ = binaryFile.Close()
			_ = os.Remove(binaryFile.Name())

			return "", err
		}
	}

	if err := binaryFile.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(binaryFile.Name(), binaryPath); err != nil {
		return "", err
	}

	return binaryPath, nil
}

func binaryPath(binaryName string) (string, error) {
	homeDir, err := homedir.Path()
	if err != nil {
		return "", err
	}

	baseDir := filepath.Join(homeDir, "cache", "bin")

	// Ensure that the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(baseDir, binaryName), nil
}
