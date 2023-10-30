package oci

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/pierrec/lz4/v4"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/regclient/manifest"
	"github.com/regclient/regclient/types"
	manifestpkg "github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"gvisor.dev/gvisor/pkg/sync"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type diskTask struct {
	Desc   types.Descriptor
	Path   string
	Offset int64
}

//nolint:gocognit // let's keep it complex for now
func PullVMDirectory(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	manifest manifest.Manifest,
	vmDir *vmdirectory.VMDirectory,
	concurrency int,
) error {
	layers, err := manifest.(manifestpkg.Imager).GetLayers()
	if err != nil {
		return err
	}

	// Find VM's config
	vmConfigs := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == MediaTypeVetuConfig
	})
	if len(vmConfigs) != 1 {
		return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
			MediaTypeVetuConfig, len(vmConfigs))
	}

	// Process VM's config
	fmt.Println("pulling config...")

	vmConfigBytes, err := pullBlob(ctx, client, reference, vmConfigs[0])
	if err != nil {
		return err
	}

	vmConfig, err := vmconfig.NewFromJSON(vmConfigBytes)
	if err != nil {
		return err
	}

	if err := vmDir.SetConfig(vmConfig); err != nil {
		return err
	}

	// Find VM's kernel
	vmKernels := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == MediaTypeVetuKernel
	})
	if len(vmKernels) != 1 {
		return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
			MediaTypeVetuKernel, len(vmKernels))
	}

	// Process VM's kernel
	fmt.Println("pulling kernel...")

	vmKernelBytes, err := pullBlob(ctx, client, reference, vmKernels[0])
	if err != nil {
		return err
	}

	if err := os.WriteFile(vmDir.KernelPath(), vmKernelBytes, 0600); err != nil {
		return err
	}

	// Find VM's initramfs
	vmInitramfses := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == MediaTypeVetuInitramfs
	})
	if len(vmInitramfses) > 0 {
		if len(vmInitramfses) > 1 {
			return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
				MediaTypeVetuInitramfs, len(vmInitramfses))
		}

		fmt.Println("pulling initramfs...")

		vmInitramfsBytes, err := pullBlob(ctx, client, reference, vmInitramfses[0])
		if err != nil {
			return err
		}

		if err := os.WriteFile(vmDir.InitramfsPath(), vmInitramfsBytes, 0600); err != nil {
			return err
		}
	}

	// Find VM's disks
	disks := lo.Filter(layers, func(desc types.Descriptor, index int) bool {
		return desc.MediaType == MediaTypeVetuDisk
	})

	// Process VM's disks by converting them into
	// disk tasks for further parallel processing
	diskTaskCh := make(chan *diskTask, len(disks))
	diskNameToOffset := map[string]int64{}

	for _, disk := range disks {
		// Extract name
		diskName, ok := disk.Annotations[AnnotationName]
		if !ok {
			return fmt.Errorf("disk layer has no %s annotation", AnnotationName)
		}

		// Name should be contained in the VM's config
		if !lo.ContainsBy(vmConfig.Disks, func(disk vmconfig.Disk) bool {
			return disk.Name == diskName
		}) {
			return fmt.Errorf("disk with name %q is not found in the VM's config", diskName)
		}

		// Extract uncompressed size
		uncompressedSizeRaw, ok := disk.Annotations[AnnotationUncompressedSize]
		if !ok {
			return fmt.Errorf("disk layer has no %s annotation", AnnotationUncompressedSize)
		}

		// Parse uncompressed size
		uncompressedSize, err := strconv.ParseInt(uncompressedSizeRaw, 10, 64)
		if err != nil {
			return err
		}

		diskTaskCh <- &diskTask{
			Desc:   disk,
			Path:   filepath.Join(vmDir.Path(), diskName),
			Offset: diskNameToOffset[diskName],
		}

		diskNameToOffset[diskName] += uncompressedSize
	}

	// There will be no more disk tasks
	close(diskTaskCh)

	// Pre-create and truncate disk files
	for diskName, offset := range diskNameToOffset {
		diskFile, err := os.OpenFile(filepath.Join(vmDir.Path(), diskName), os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		if err := diskFile.Truncate(offset); err != nil {
			return err
		}

		if err := diskFile.Close(); err != nil {
			return err
		}
	}

	// Process disk tasks with the specified concurrency
	totalUncompressedDisksSizeBytes := lo.Sum(lo.Values(diskNameToOffset))
	totalCompressedDisksSizeBytes := lo.Sum(lo.Map(disks, func(diskDesc types.Descriptor, index int) int64 {
		return diskDesc.Size
	}))
	fmt.Printf("pulling %d disk(s) (%s compressed, %s uncompressed)...\n", len(diskNameToOffset),
		humanize.Bytes(uint64(totalCompressedDisksSizeBytes)),
		humanize.Bytes(uint64(totalUncompressedDisksSizeBytes)))

	progressBar := progressbar.DefaultBytes(totalCompressedDisksSizeBytes)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	diskTasksErrCh := make(chan error, concurrency)

	diskTasksCtx, diskTasksCtxCancel := context.WithCancel(ctx)
	defer diskTasksCtxCancel()

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			for {
				diskTask, ok := <-diskTaskCh
				if !ok {
					return
				}

				if err := diskTask.process(diskTasksCtx, client, reference, progressBar); err != nil {
					diskTasksErrCh <- err
					diskTasksCtxCancel()
				}
			}
		}()
	}

	// Wait for the disk tasks to finish
	wg.Wait()

	// Since we've finished with pulling disks,
	// we can finish the associated progress bar
	if err := progressBar.Finish(); err != nil {
		return err
	}

	// Check for errors
	select {
	case err := <-diskTasksErrCh:
		return err
	default:
		// no error reported
	}

	return nil
}

func pullBlob(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	descriptor types.Descriptor,
) ([]byte, error) {
	blobReader, err := client.BlobGet(ctx, reference, descriptor)
	if err != nil {
		return nil, err
	}
	defer blobReader.Close()

	return io.ReadAll(blobReader)
}

func (diskTask *diskTask) process(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	progressBar *progressbar.ProgressBar,
) error {
	// Open disk file and seek to the specified offset
	diskFile, err := os.OpenFile(diskTask.Path, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := diskFile.Seek(diskTask.Offset, io.SeekStart); err != nil {
		return err
	}

	// Pull disk layer from the OCI registry
	blobReader, err := client.BlobGet(ctx, reference, diskTask.Desc)
	if err != nil {
		return err
	}
	defer blobReader.Close()

	// Decompress the disk data on-the-fly and write it to the disk file
	progressBarReader := progressbar.NewReader(blobReader, progressBar)
	lz4Reader := lz4.NewReader(&progressBarReader)

	if _, err := io.Copy(diskFile, lz4Reader); err != nil {
		return err
	}

	if err := diskFile.Close(); err != nil {
		return err
	}

	return nil
}
