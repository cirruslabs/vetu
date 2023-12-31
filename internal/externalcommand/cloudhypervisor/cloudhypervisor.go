package cloudhypervisor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/binaryfetcher"
	"io"
	"os/exec"
	"runtime"
)

const (
	binaryName = "cloud-hypervisor"
	baseURL    = "https://github.com/cloud-hypervisor/cloud-hypervisor/releases/latest/download/"
)

var goarchToDownloadURL = map[string]string{
	"amd64": baseURL + "cloud-hypervisor-static",
	"arm64": baseURL + "cloud-hypervisor-static-aarch64",
}

func CloudHypervisor(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Always prefer the Cloud Hypervisor binary in PATH
	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		// Fall back to downloading the Cloud Hypervisor binary from GitHub
		downloadURL, ok := goarchToDownloadURL[runtime.GOARCH]
		if !ok {
			return nil, fmt.Errorf("no %q binary found in PATH and architecture %q "+
				"is not available in Cloud Hypervisor's GitHub releases", binaryName, runtime.GOARCH)
		}

		fmt.Printf("no %q binary found in PATH, downloading it from %s...\n", binaryName, downloadURL)

		binaryPath, err = binaryfetcher.GetOrFetch(ctx, func(ctx context.Context, binaryFile io.Writer) error {
			// Download the Cloud Hypervisor binary if not available in the cache
			resp, err := binaryfetcher.FetchURL(ctx, downloadURL)
			if err != nil {
				return err
			}

			if _, err := io.Copy(binaryFile, resp.Body); err != nil {
				return err
			}

			return nil
		}, binaryName, true)
		if err != nil {
			return nil, err
		}
	}

	return exec.CommandContext(ctx, binaryPath, args...), nil
}
