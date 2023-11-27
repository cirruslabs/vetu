package remote

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/dockerhosts"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/oci"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

func Pull(ctx context.Context, remoteName remotename.RemoteName, insecure bool, concurrency int) error {
	// Convert remoteName to ref.Ref that is used in github.com/regclient/regclient
	reference, err := ref.New(remoteName.String())
	if err != nil {
		return err
	}

	// Initialize a temporary directory to which we'll first pull the VM image
	vmDir, err := temporary.Create()
	if err != nil {
		return err
	}

	// Load hosts from the Docker configuration file
	hosts, err := dockerhosts.Load(reference, insecure)
	if err != nil {
		return err
	}

	// Initialize OCI registry client
	client := regclient.New(regclient.WithConfigHost(hosts...))

	// Resolve the reference to a manifest
	fmt.Printf("pulling %s...\n", reference.CommonName())

	fmt.Println("pulling manifest...")

	manifest, err := client.ManifestGet(ctx, reference)
	if err != nil {
		return err
	}

	// Make the remote name that we've got from the user fully qualified
	// by stripping the tag and setting its digest to the manifest's digest
	fullyQualifiedRemoteName := remoteName
	fullyQualifiedRemoteName.Tag = ""
	fullyQualifiedRemoteName.Digest = manifest.GetDescriptor().Digest

	// Pull the VM image if we don't have one already in cache
	if !Exists(fullyQualifiedRemoteName) {
		if err := oci.PullVMDirectory(ctx, client, reference, manifest, vmDir, concurrency); err != nil {
			return err
		}

		// We've successfully pulled the VM image, we can now atomically move
		// the temporary directory containing it to its final destination
		return MoveIn(remoteName, manifest.GetDescriptor().Digest, vmDir)
	} else {
		fmt.Printf("skipping pull because %s already exists in the OCI cache...\n",
			fullyQualifiedRemoteName)
	}

	// Link the remote names if tag is used and no linkage already exists in the storage
	if remoteName.Tag != "" && !Exists(remoteName) {
		fmt.Printf("creating a link from %s to %s...\n", remoteName, fullyQualifiedRemoteName)

		if err := Link(fullyQualifiedRemoteName, remoteName); err != nil {
			return err
		}
	}

	// If the digest was used when pulling this VM image, mark it as
	// explicitly pulled to prevent the automatic garbage collection
	if remoteName.Digest != "" {
		vmDir, err := Open(remoteName)
		if err != nil {
			return err
		}

		if err := vmDir.SetExplicitlyPulled(true); err != nil {
			return err
		}
	}

	return nil
}
