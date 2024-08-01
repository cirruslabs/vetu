package pullhelper

import (
	"context"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/ref"
	"io"
)

func PullBlob(
	ctx context.Context,
	client *regclient.RegClient,
	reference ref.Ref,
	descriptor descriptor.Descriptor,
) ([]byte, error) {
	blobReader, err := client.BlobGet(ctx, reference, descriptor)
	if err != nil {
		return nil, err
	}
	defer blobReader.Close()

	return io.ReadAll(blobReader)
}
