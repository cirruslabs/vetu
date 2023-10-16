package pull

import (
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/oci"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
)

var concurrency uint8

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull REMOTE_NAME",
		Short: "Pull a VM from a registry",
		RunE:  runPull,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Uint8Var(&concurrency, "concurrency", 4,
		"network concurrency to use when pulling a remote VM from the OCI-compatible registry")

	return cmd
}

func runPull(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Parse name
	remoteName, err := remotename.NewFromString(name)
	if err != nil {
		return err
	}

	// Initialize a temporary directory to which we'll first pull the VM image
	vmDir, err := temporary.Create()
	if err != nil {
		return err
	}

	// Initialize OCI registry client and convert remote name to a reference
	client := regclient.New(regclient.WithDockerCreds())

	// Convert remoteName to ref.Ref that is used in github.com/regclient/regclient
	reference, err := ref.New(remoteName.String())
	if err != nil {
		return err
	}

	// Pull the VM image
	digest, err := oci.PullVMDirectory(cmd.Context(), client, reference, vmDir, int(concurrency))
	if err != nil {
		return err
	}

	// We've successfully pulled the VM image, we can now atomically move
	// the temporary directory containing it to its final destination
	return remote.MoveIn(remoteName, digest, vmDir)
}
