package pull

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/dockerhosts"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/oci"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
)

var concurrency uint8
var insecure bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull REMOTE_NAME",
		Short: "Pull a VM from a registry",
		RunE:  runPull,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Uint8Var(&concurrency, "concurrency", 4,
		"network concurrency to use when pulling a remote VM from the OCI-compatible registry")
	cmd.Flags().BoolVar(&insecure, "insecure", false,
		"connect to the OCI registry via insecure HTTP protocol")

	return cmd
}

func runPull(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Parse name
	remoteName, err := remotename.NewFromString(name)
	if err != nil {
		if errors.Is(err, remotename.ErrNotARemoteName) {
			fmt.Printf("%q is a local image, nothing to pull here!\n", name)

			return nil
		}

		return err
	}

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

	manifest, err := client.ManifestGet(cmd.Context(), reference)
	if err != nil {
		return err
	}

	// Make the remote name that we've got from the user fully qualified
	// by stripping the tag and setting its digest to the manifest's digest
	fullyQualifiedRemoteName := remoteName
	fullyQualifiedRemoteName.Tag = ""
	fullyQualifiedRemoteName.Digest = manifest.GetDescriptor().Digest

	// Pull the VM image if we don't have one already in cache
	if !remote.Exists(fullyQualifiedRemoteName) {
		if err := oci.PullVMDirectory(cmd.Context(), client, reference, manifest, vmDir, int(concurrency)); err != nil {
			return err
		}

		// We've successfully pulled the VM image, we can now atomically move
		// the temporary directory containing it to its final destination
		return remote.MoveIn(remoteName, manifest.GetDescriptor().Digest, vmDir)
	} else {
		fmt.Printf("skipping pull because %s already exists in the OCI cache...\n",
			fullyQualifiedRemoteName)
	}

	// Link the remote names if tag is used and no linkage already exists in the storage
	if remoteName.Tag != "" && !remote.Exists(remoteName) {
		fmt.Printf("creating a link from %s to %s...\n", remoteName, fullyQualifiedRemoteName)

		if err := remote.Link(fullyQualifiedRemoteName, remoteName); err != nil {
			return err
		}
	}

	// If the digest was used when pulling this VM image, mark it as
	// explicitly pulled to prevent the automatic garbage collection
	if remoteName.Digest != "" {
		vmDir, err := remote.Open(remoteName)
		if err != nil {
			return err
		}

		if err := vmDir.SetExplicitlyPulled(true); err != nil {
			return err
		}
	}

	return nil
}
