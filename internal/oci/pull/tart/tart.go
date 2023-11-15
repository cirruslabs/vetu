package tart

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/oci/annotations"
	"github.com/cirruslabs/vetu/internal/oci/diskpuller"
	"github.com/cirruslabs/vetu/internal/oci/mediatypes"
	"github.com/cirruslabs/vetu/internal/oci/pull/pullhelper"
	"github.com/cirruslabs/vetu/internal/oci/pull/tart/applestream"
	"github.com/cirruslabs/vetu/internal/oci/pull/tart/tartconfig"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	cp "github.com/otiai10/copy"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	manifestpkg "github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
	"io"
)

const diskName = "disk.img"

func PullVMDirectory(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	manifest manifestpkg.Manifest,
	vmDir *vmdirectory.VMDirectory,
	concurrency int,
) error {
	// Get layers
	layers, err := manifest.(manifestpkg.Imager).GetLayers()
	if err != nil {
		return err
	}

	// Find VM's config
	vmConfigs := lo.Filter(layers, func(descriptor types.Descriptor, index int) bool {
		return descriptor.MediaType == mediatypes.MediaTypeTartConfig
	})
	if len(vmConfigs) != 1 {
		return fmt.Errorf("manifest should contain exactly one layer of type %s, found %d",
			mediatypes.MediaTypeTartConfig, len(vmConfigs))
	}

	// Pull and process Tart VM's config
	fmt.Println("pulling config...")

	tartConfigBytes, err := pullhelper.PullBlob(ctx, client, reference, vmConfigs[0])
	if err != nil {
		return err
	}

	tartConfig, err := tartconfig.NewFromJSON(tartConfigBytes)
	if err != nil {
		return err
	}

	if tartConfig.OS != "linux" {
		return fmt.Errorf("you're attempting to pull a Tart VM that's built for %q, "+
			"but only \"linux\" is currently supported", tartConfig.OS)
	}

	// Create a Vetu-native VM config
	vmConfig := vmconfig.New()

	vmConfig.Arch = tartConfig.Arch
	vmConfig.Disks = []vmconfig.Disk{
		{Name: diskName},
	}
	vmConfig.CPUCount = tartConfig.CPUCount
	vmConfig.MemorySize = tartConfig.MemorySize
	vmConfig.MACAddress = tartConfig.MACAddress

	if err := vmDir.SetConfig(vmConfig); err != nil {
		return err
	}

	// Copy latest Hypervisor Firmware since
	// Tart VM images have no separate kernel
	fmt.Println("copying EDK2 firmware to use as a kernel...")

	if err := cp.Copy("/usr/share/cloud-hypervisor/CLOUDHV_EFI.fd", vmDir.KernelPath()); err != nil {
		return err
	}

	// Find VM's disks
	disks := lo.Filter(layers, func(desc types.Descriptor, index int) bool {
		return desc.MediaType == mediatypes.MediaTypeTartDisk
	})

	// Pull VM's disks
	nameFunc := func(disk types.Descriptor) (string, error) {
		// Tart VM images have only one disk
		return diskName, nil
	}

	decompressorFunc := func(r io.Reader) io.Reader {
		return applestream.NewReader(r)
	}

	return diskpuller.PullDisks(ctx, client, reference, vmDir, concurrency, disks, nameFunc,
		annotations.AnnotationTartUncompressedSize, decompressorFunc)
}
