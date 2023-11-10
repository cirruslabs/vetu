package hypervisorfw

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

const latestURL = "https://github.com/cloud-hypervisor/rust-hypervisor-firmware/releases/latest/download/hypervisor-fw"

func Fetch(ctx context.Context, destPath string) error {
	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch Rust Hypervisor Firmware: HTTP %d",
			resp.StatusCode)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(destFile, resp.Body); err != nil {
		return err
	}

	return nil
}
