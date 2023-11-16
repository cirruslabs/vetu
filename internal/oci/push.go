package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	chunkerpkg "github.com/cirruslabs/vetu/internal/chunker"
	"github.com/cirruslabs/vetu/internal/oci/annotations"
	"github.com/cirruslabs/vetu/internal/oci/mediatypes"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/opencontainers/go-digest"
	"github.com/pierrec/lz4/v4"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/blob"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const targetDiskLayerSizeBytes = 500 * humanize.MByte

func PushVMDirectory(
	ctx context.Context,
	client *regclient.RegClient,
	vmDir *vmdirectory.VMDirectory,
	reference ref.Ref,
) (digest.Digest, error) {
	fmt.Printf("pushing %s...\n", reference.CommonName())

	// Create an OCI image manifest
	ociManifest := v1.Manifest{
		Versioned: v1.ManifestSchemaVersion,
		MediaType: types.MediaTypeOCI1Manifest,
	}

	// Create an OCI image configuration
	ociConfig := blob.NewOCIConfig(blob.WithImage(v1.Image{
		Platform: platform.Platform{
			Architecture: runtime.GOARCH,
			OS:           runtime.GOOS,
		},
	}))

	// Push the OCI image configuration and add
	// its descriptor to the OCI image manifest
	var err error

	ociManifest.Config, err = pushJSON(ctx, client, reference, ociConfig,
		types.MediaTypeOCI1ImageConfig, nil)
	if err != nil {
		return "", err
	}

	// Push VM's config
	fmt.Println("pushing config...")

	vmConfigDesc, err := pushFile(ctx, client, reference, vmDir.ConfigPath(),
		mediatypes.MediaTypeConfig, nil)
	if err != nil {
		return "", err
	}
	ociManifest.Layers = append(ociManifest.Layers, vmConfigDesc)

	// Push VM's kernel
	fmt.Println("pushing kernel...")

	vmKernelDesc, err := pushFile(ctx, client, reference, vmDir.KernelPath(),
		mediatypes.MediaTypeKernel, nil)
	if err != nil {
		return "", err
	}
	ociManifest.Layers = append(ociManifest.Layers, vmKernelDesc)

	// Push VM's initramfs (if any)
	initramfsBytes, err := os.ReadFile(vmDir.InitramfsPath())
	if err != nil {
		// Report an error if the initramfs exists,
		// but we cannot access it for some reason
		if !os.IsNotExist(err) {
			return "", err
		}
	} else {
		fmt.Println("pushing initramfs...")

		vmInitramfsDesc, err := pushBytes(ctx, client, reference, initramfsBytes,
			mediatypes.MediaTypeInitramfs, nil, nil)
		if err != nil {
			return "", err
		}

		ociManifest.Layers = append(ociManifest.Layers, vmInitramfsDesc)
	}

	// Push VM's disks
	for _, disk := range vmDir.Config().Disks {
		fmt.Printf("pushing disk %s...\n", disk.Name)

		vmDiskDescriptors, err := pushDisk(ctx, client, reference, filepath.Join(vmDir.Path(), disk.Name),
			disk.Name)
		if err != nil {
			return "", err
		}

		ociManifest.Layers = append(ociManifest.Layers, vmDiskDescriptors...)
	}

	m, err := manifest.New(manifest.WithOrig(ociManifest))
	if err != nil {
		return "", err
	}

	if err := client.ManifestPut(ctx, reference, m); err != nil {
		return "", err
	}

	return m.GetDescriptor().Digest, nil
}

func pushFile(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	path string,
	mediaType string,
	annotations map[string]string,
) (types.Descriptor, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return types.Descriptor{}, err
	}

	return pushBytes(ctx, client, reference, fileBytes, mediaType, annotations, nil)
}

func pushJSON(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	obj interface{},
	mediaType string,
	annotations map[string]string,
) (types.Descriptor, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return types.Descriptor{}, err
	}

	return pushBytes(ctx, client, reference, jsonBytes, mediaType, annotations, nil)
}

func pushBytes(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	data []byte,
	mediaType string,
	annotations map[string]string,
	progressBar *progressbar.ProgressBar,
) (types.Descriptor, error) {
	desc := types.Descriptor{
		MediaType:   mediaType,
		Size:        int64(len(data)),
		Digest:      digest.FromBytes(data),
		Annotations: annotations,
	}

	var reader io.Reader

	if progressBar != nil {
		// Wrap in progress bar reader,
		// if progress bar is supplied
		progressBarReader := progressbar.NewReader(bytes.NewReader(data), progressBar)
		reader = &progressBarReader
	} else {
		// Use plain reader
		reader = bytes.NewReader(data)
	}

	return client.BlobPut(ctx, reference, desc, reader)
}

func pushDisk(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	path string,
	diskName string,
) ([]types.Descriptor, error) {
	var result []types.Descriptor

	diskFile, err := os.Open(path)
	if err != nil {
		return []types.Descriptor{}, err
	}

	chunker := chunkerpkg.NewChunker(targetDiskLayerSizeBytes, func(w io.Writer) (io.WriteCloser, error) {
		return lz4.NewWriter(w), nil
	})

	errCh := make(chan error, 1)

	go func() {
		if _, err := io.Copy(chunker, diskFile); err != nil {
			errCh <- err

			return
		}

		if err := chunker.Close(); err != nil {
			errCh <- err

			return
		}

		errCh <- nil
	}()

	progressBar := progressbar.DefaultBytes(-1)

	for compressedChunk := range chunker.Chunks() {
		annotations := map[string]string{
			annotations.AnnotationName:               diskName,
			annotations.AnnotationUncompressedSize:   strconv.FormatInt(compressedChunk.UncompressedSize, 10),
			annotations.AnnotationUncompressedDigest: compressedChunk.UncompressedDigest.String(),
		}

		diskDesc, err := pushBytes(ctx, client, reference, compressedChunk.Data,
			mediatypes.MediaTypeDisk, annotations, progressBar)
		if err != nil {
			return nil, err
		}

		result = append(result, diskDesc)
	}

	// Since we've finished pushing the disk,
	// we can finish the associated progress bar
	if err := progressBar.Finish(); err != nil {
		return nil, err
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	return result, nil
}
