package vetu

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/oci/annotations"
	"github.com/cirruslabs/vetu/internal/oci/diskpuller"
	"github.com/cirruslabs/vetu/internal/oci/mediatypes"
	"github.com/cirruslabs/vetu/internal/oci/pull/pullhelper"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/pierrec/lz4/v4"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	manifestpkg "github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
	"io"
	"os"
)

func PullVMDirectory(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	manifest manifestpkg.Manifest,
	vmDir *vmdirectory.VMDirectory,
	concurrency int,
) error {
	layers, err := manifest.(manifestpkg.Imager).GetLayers()
	if err != nil {
		return err
	}

	// Find VM's config
	vmConfigs := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == mediatypes.MediaTypeConfig
	})
	if len(vmConfigs) != 1 {
		return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
			mediatypes.MediaTypeConfig, len(vmConfigs))
	}

	// Process VM's config
	fmt.Println("pulling config...")

	vmConfigBytes, err := pullhelper.PullBlob(ctx, client, reference, vmConfigs[0])
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
		return descriptor.MediaType == mediatypes.MediaTypeKernel
	})
	if len(vmKernels) != 1 {
		return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
			mediatypes.MediaTypeKernel, len(vmKernels))
	}

	// Process VM's kernel
	fmt.Println("pulling kernel...")

	vmKernelBytes, err := pullhelper.PullBlob(ctx, client, reference, vmKernels[0])
	if err != nil {
		return err
	}

	if err := os.WriteFile(vmDir.KernelPath(), vmKernelBytes, 0600); err != nil {
		return err
	}

	// Find VM's initramfs
	vmInitramfses := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == mediatypes.MediaTypeInitramfs
	})
	if len(vmInitramfses) > 0 {
		if len(vmInitramfses) > 1 {
			return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
				mediatypes.MediaTypeInitramfs, len(vmInitramfses))
		}

		fmt.Println("pulling initramfs...")

		vmInitramfsBytes, err := pullhelper.PullBlob(ctx, client, reference, vmInitramfses[0])
		if err != nil {
			return err
		}

		if err := os.WriteFile(vmDir.InitramfsPath(), vmInitramfsBytes, 0600); err != nil {
			return err
		}
	}

	// Find VM's disks
	disks := lo.Filter(layers, func(desc types.Descriptor, index int) bool {
		return desc.MediaType == mediatypes.MediaTypeDisk
	})

	// Pull VM's disks
	nameFunc := func(disk types.Descriptor) (string, error) {
		// Extract name
		diskName, ok := disk.Annotations[annotations.AnnotationName]
		if !ok {
			return "", fmt.Errorf("disk layer has no %s annotation", annotations.AnnotationName)
		}

		// Name should be contained in the VM's config
		if !lo.ContainsBy(vmConfig.Disks, func(disk vmconfig.Disk) bool {
			return disk.Name == diskName
		}) {
			return "", fmt.Errorf("disk with name %q is not found in the VM's config", diskName)
		}

		return diskName, nil
	}

	decompressorFunc := func(r io.Reader) io.Reader {
		return lz4.NewReader(r)
	}

	return diskpuller.PullDisks(ctx, client, reference, vmDir, concurrency, disks, nameFunc,
		annotations.AnnotationUncompressedSize, decompressorFunc)
}
