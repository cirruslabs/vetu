//go:build !(linux && amd64)

package firmware

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/binaryfetcher"
	"github.com/samber/lo"
	"io"
	"os"
	"path"
	"pault.ag/go/debian/control"
	"pault.ag/go/debian/deb"
	"runtime"
)

const (
	systemEDKPath = "/usr/share/cloud-hypervisor/CLOUDHV_EFI.fd"

	debRepositoryURL  = "https://download.opensuse.org/repositories/home:/cloud-hypervisor/xUbuntu_22.04"
	debTargetPackage  = "edk2-cloud-hypervisor"
	debTargetFilename = "CLOUDHV_EFI.fd"
)

func Firmware(ctx context.Context) (string, string, error) {
	// Always prefer the EDK2 firmware installed on the system
	_, err := os.Stat(systemEDKPath)
	if err == nil {
		return systemEDKPath, "EDK2 firmware", nil
	}

	// Fall back to downloading the EDK2 firmware from a .deb-repository
	fmt.Printf("no EDK2 firmware installed on the system, downloading it from %s...\n",
		debRepositoryURL)

	binaryPath, err := binaryfetcher.GetOrFetch(ctx, func(ctx context.Context, binaryFile io.Writer) error {
		// Fetch the Packages file to determine the appropriate .deb
		// that'll run on runtime.GOARCH
		debURL, err := determineDebURL(ctx)
		if err != nil {
			return err
		}

		// Fetch the .deb file and extract the firmware contents to binaryFile
		return downloadAndExtractDeb(ctx, debURL, binaryFile)
	}, debTargetFilename, true)
	if err != nil {
		return "", "", err
	}

	return binaryPath, "downloaded EDK2 firmware", nil
}

func determineDebURL(ctx context.Context) (string, error) {
	// Fetch the Packages file and parse it
	resp, err := binaryfetcher.FetchURL(ctx, debRepositoryURL+"/Packages")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	sources, err := control.ParseBinaryIndex(bufio.NewReader(resp.Body))
	if err != nil {
		return "", err
	}

	// Find the package that contains EDK2 firmware for the current architecture
	edk2Source, ok := lo.Find(sources, func(source control.BinaryIndex) bool {
		return source.Package == debTargetPackage && source.Architecture.CPU == runtime.GOARCH
	})
	if !ok {
		return "", fmt.Errorf("cannot find edk2-cloud-hypervisor package for %v in the repository",
			runtime.GOARCH)
	}

	return debRepositoryURL + "/" + edk2Source.Filename, nil
}

func downloadAndExtractDeb(ctx context.Context, debURL string, binaryFile io.Writer) error {
	// Fetch the .deb package and parse it
	debPath, err := binaryfetcher.FetchURLToFile(ctx, debURL)
	if err != nil {
		return err
	}
	defer os.Remove(debPath)

	parsedDeb, debCloser, err := deb.LoadFile(debPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = debCloser()
	}()

	// Iterate over .deb package data files and look for EDK2 firmware
	for {
		next, err := parsedDeb.Data.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("cannot find %s file in the %s package", debTargetFilename,
					debURL)
			}

			return err
		}

		if path.Base(next.Name) == debTargetFilename {
			_, err := io.Copy(binaryFile, parsedDeb.Data)

			return err
		}
	}
}
