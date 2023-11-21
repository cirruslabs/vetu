package firmware

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/binaryfetcher"
	"os"
	"path"
	"runtime"
)

const (
	edk2BinaryPath = "/usr/share/cloud-hypervisor/CLOUDHV_EFI.fd"
	baseURL        = "https://github.com/cirruslabs/rust-hypervisor-firmware/releases/latest/download"
)

var goarchToDownloadURL = map[string]string{
	"amd64": path.Join(baseURL, "hypervisor-fw"),
	"arm64": path.Join(baseURL, "hypervisor-fw-aarch64"),
}

func Firmware(ctx context.Context) (string, string, error) {
	// Always prefer the EDK2 firmware installed on the system
	_, err := os.Stat(edk2BinaryPath)
	if err == nil {
		return edk2BinaryPath, "EDK2 firmware", nil
	}

	// Fall back to downloading the Rust Hypervisor Firmware from GitHub
	downloadURL, ok := goarchToDownloadURL[runtime.GOARCH]
	if !ok {
		return "", "", fmt.Errorf("no EDK2 firmware installed on the system "+
			"and architecture %q is not available in Rust Hypervisor Firmware's GitHub releases", runtime.GOARCH)
	}

	fmt.Printf("no EDK2 firmware installed on the system, downloading Rust Hypervisor Firmware "+
		"from %s...\n", downloadURL)

	binaryPath, err := binaryfetcher.Fetch(ctx, downloadURL, "hypervisor-fw", true)
	if err != nil {
		return "", "", err
	}

	return binaryPath, "Rust Hypervisor Firmware", nil
}
