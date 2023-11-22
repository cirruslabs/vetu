package push

import (
	"github.com/cirruslabs/vetu/internal/dockerhosts"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/oci"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
)

var populateCache bool
var insecure bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push LOCAL_NAME REMOTE_NAME",
		Short: "Push a VM to a registry",
		RunE:  runPush,
		Args:  cobra.ExactArgs(2),
	}

	cmd.Flags().BoolVar(&populateCache, "populate-cache", false, "cache pushed image locally, "+
		"increases disk usage, but saves time if you're going to pull the pushed images shortly thereafter")
	cmd.Flags().BoolVar(&insecure, "insecure", false,
		"connect to the OCI registry via insecure HTTP protocol")

	return cmd
}

func runPush(cmd *cobra.Command, args []string) error {
	srcName := args[0]
	dstName := args[1]

	// Parse srcName
	srcLocalName, err := localname.NewFromString(srcName)
	if err != nil {
		return err
	}

	vmDir, err := local.Open(srcLocalName)
	if err != nil {
		return err
	}

	// Parse dstName
	dstRemoteName, err := remotename.NewFromString(dstName)
	if err != nil {
		return err
	}

	// Convert dstRemoteName to ref.Ref that is used in github.com/regclient/regclient
	reference, err := ref.New(dstRemoteName.String())
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

	// Push the VM image
	digest, err := oci.PushVMDirectory(cmd.Context(), client, vmDir, reference)
	if err != nil {
		return err
	}

	// If requested, cache the pushed VM image locally
	if populateCache {
		return remote.MoveIn(dstRemoteName, digest, vmDir)
	}

	return nil
}
