package pull

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/storage/remote"
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

	return remote.Pull(cmd.Context(), remoteName, insecure, int(concurrency))
}
