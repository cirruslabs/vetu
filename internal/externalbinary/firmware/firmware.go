package firmware

import (
	"context"
	"github.com/cirruslabs/vetu/internal/binaryfetcher"
	"io"
	"strings"
)

// Unfortunately, the EDK2 version distributed through package repositories[1]
// is not compatible with --device (and to be precisely, GPU passthrough),
// so we fetch the latest EDK2 version from Cloud Hypervisor's GitHub repo
// to work around the VM booting issues when a GPU with large memory
// is attached[2][3].
//
// [1]: https://download.opensuse.org/repositories/home:/cloud-hypervisor/
// [2]: https://edk2.groups.io/g/discuss/topic/59340711
// [3]: https://github.com/cloud-hypervisor/cloud-hypervisor/issues/6147
const (
	githubURL      = "https://github.com/cloud-hypervisor/edk2/releases/download/ch-6624aa331f/CLOUDHV.fd"
	githubFilename = "CLOUDHV-6624aa331f.fd"
)

func Firmware(ctx context.Context) (string, string, error) {
	desc := "cached EDK2 firmware from GitHub"

	path, err := binaryfetcher.GetOrFetch(ctx, func(ctx context.Context, binaryFile io.Writer) error {
		resp, err := binaryfetcher.FetchURL(ctx, githubURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		desc = strings.ReplaceAll(desc, "cached", "downloaded")

		_, err = io.Copy(binaryFile, resp.Body)

		return err
	}, githubFilename, true)
	if err != nil {
		return "", "", err
	}

	return path, desc, nil
}
