package oci

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/oci/mediatypes"
	"github.com/cirruslabs/vetu/internal/oci/pull/tart"
	"github.com/cirruslabs/vetu/internal/oci/pull/vetu"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/regclient/manifest"
	"github.com/regclient/regclient/types"
	manifestpkg "github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/samber/lo"
)

func PullVMDirectory(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	manifest manifest.Manifest,
	vmDir *vmdirectory.VMDirectory,
	concurrency int,
) error {
	// Get layers
	layers, err := manifest.(manifestpkg.Imager).GetLayers()
	if err != nil {
		return err
	}

	// Determine the VM image type
	mediaTypes := lo.Map(layers, func(layer types.Descriptor, index int) string {
		return layer.MediaType
	})

	switch {
	case lo.Contains(mediaTypes, mediatypes.MediaTypeConfig):
		return vetu.PullVMDirectory(ctx, client, reference, manifest, vmDir, concurrency)
	case lo.Contains(mediaTypes, mediatypes.MediaTypeTartConfig):
		return tart.PullVMDirectory(ctx, client, reference, manifest, vmDir, concurrency)
	default:
		return fmt.Errorf("unsupported VM image type")
	}
}
