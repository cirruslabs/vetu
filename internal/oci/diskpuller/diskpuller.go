package diskpuller

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type NameFromDiskDescriptorFunc func(diskDescriptor types.Descriptor) (string, error)
type InitializeDecompressorFunc func(compressedReader io.Reader) io.Reader

type diskTask struct {
	Desc   types.Descriptor
	Path   string
	Offset int64
}

func PullDisks(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	vmDir *vmdirectory.VMDirectory,
	concurrency int,
	disks []types.Descriptor,
	nameFromDiskDescriptor NameFromDiskDescriptorFunc,
	uncompressedSizeAnnotation string,
	initializeDecompressor InitializeDecompressorFunc,
) error {
	// Process VM's disks by converting them into
	// disk tasks for further parallel processing
	diskTaskCh := make(chan *diskTask, len(disks))
	diskNameToOffset := map[string]int64{}

	for _, disk := range disks {
		// Extract name
		diskName, err := nameFromDiskDescriptor(disk)
		if err != nil {
			return err
		}

		// Extract and parse uncompressed size
		uncompressedSizeRaw, ok := disk.Annotations[uncompressedSizeAnnotation]
		if !ok {
			return fmt.Errorf("disk layer has no %s annotation", uncompressedSizeAnnotation)
		}
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

				if err := diskTask.process(diskTasksCtx, client, reference, progressBar, initializeDecompressor); err != nil {
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
		return nil
	}
}

func (diskTask *diskTask) process(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	progressBar *progressbar.ProgressBar,
	initializeDecompressor InitializeDecompressorFunc,
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
	decompressor := initializeDecompressor(&progressBarReader)

	if _, err := io.Copy(diskFile, decompressor); err != nil {
		return err
	}

	return diskFile.Close()
}
