package tart

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/oci/annotations"
	"github.com/cirruslabs/vetu/internal/oci/diskpuller"
	"github.com/cirruslabs/vetu/internal/oci/mediatypes"
	"github.com/cirruslabs/vetu/internal/oci/pull/pullhelper"
	"github.com/cirruslabs/vetu/internal/oci/pull/tart/applestream"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
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

	// Inject a single disk
	vmConfig.Disks = []vmconfig.Disk{
		{
			Name: diskName,
		},
	}

	if err := vmDir.SetConfig(vmConfig); err != nil {
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
