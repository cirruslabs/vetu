package binaryfetcher

import (
	"context"
	"github.com/cirruslabs/vetu/internal/homedir"
	"io"
	"os"
	"path/filepath"
)

type FetchFunc func(ctx context.Context, binaryFile io.Writer) error

func Fetch(ctx context.Context, fetchFunc FetchFunc, binaryName string, executable bool) (string, error) {
	// Determine the binary path
	binaryPath, err := binaryPath(binaryName)
	if err != nil {
		return "", err
	}

	// Use the cached binary if possible
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Run the user-provided function to fetch the binary file
	// if not available in the cache
	binaryFile, err := os.Create(binaryPath)
	if err != nil {
		return "", err
	}
	defer binaryFile.Close()

	if err := fetchFunc(ctx, binaryFile); err != nil {
		return "", err
	}

	// Make the binary executable if requested
	if executable {
		if err := binaryFile.Chmod(0755); err != nil {
			return "", err
		}
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
