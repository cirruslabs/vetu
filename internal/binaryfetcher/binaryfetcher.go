package binaryfetcher

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/homedir"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func Fetch(ctx context.Context, downloadURL string, binaryName string, executable bool) (string, error) {
	// Determine the binary path
	binaryPath, err := binaryPath(binaryName)
	if err != nil {
		return "", err
	}

	// Use the cached binary if possible
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Download and cache the binary if not available in the cache
	client := http.Client{}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %q binary from %s: HTTP %d",
			binaryName, downloadURL, resp.StatusCode)
	}

	binaryFile, err := os.Create(binaryPath)
	if err != nil {
		return "", err
	}
	defer binaryFile.Close()

	if _, err := io.Copy(binaryFile, resp.Body); err != nil {
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
