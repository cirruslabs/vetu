package clone

import (
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/randommac"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/spf13/cobra"
)

var concurrency uint8
var insecure bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone NAME LOCAL_NAME",
		Short: "Clone a VM",
		RunE:  runClone,
		Args:  cobra.ExactArgs(2),
	}

	cmd.Flags().Uint8Var(&concurrency, "concurrency", 4,
		"network concurrency to use when pulling a remote VM from the OCI-compatible registry")
	cmd.Flags().BoolVar(&insecure, "insecure", false,
		"connect to the OCI registry via insecure HTTP protocol")

	return cmd
}

func runClone(cmd *cobra.Command, args []string) error {
	srcNameRaw := args[0]
	dstNameRaw := args[1]

	srcName, err := name.NewFromString(srcNameRaw)
	if err != nil {
		return err
	}

	dstLocalName, err := localname.NewFromString(dstNameRaw)
	if err != nil {
		return err
	}

	// Check if we need to pull anything
	remoteName, ok := srcName.(remotename.RemoteName)
	if ok && !remote.Exists(remoteName) {
		if err := remote.Pull(cmd.Context(), remoteName, insecure, int(concurrency)); err != nil {
			return err
		}
	}

	// Open and lock the source VM directory under a global lock
	srcVMDir, err := globallock.With(cmd.Context(), func() (*vmdirectory.VMDirectory, error) {
		var srcVMDir *vmdirectory.VMDirectory

		switch typedSrcName := srcName.(type) {
		case localname.LocalName:
			srcVMDir, err = local.Open(typedSrcName)
		case remotename.RemoteName:
			srcVMDir, err = remote.Open(typedSrcName)
		}
		if err != nil {
			return nil, err
		}

		lock, err := srcVMDir.FileLock(filelock.LockShared)
		if err != nil {
			return nil, err
		}

		if err := lock.Trylock(); err != nil {
			return nil, err
		}

		return srcVMDir, nil
	})
	if err != nil {
		return err
	}

	tmpVMDir, err := temporary.CreateFrom(srcVMDir.Path())
	if err != nil {
		return err
	}

	// Generate and set a random MAC-address
	vmConfig, err := tmpVMDir.Config()
	if err != nil {
		return err
	}
	vmConfig.MACAddress.HardwareAddr, err = randommac.UnicastAndLocallyAdministered()
	if err != nil {
		return err
	}
	if err := tmpVMDir.SetConfig(vmConfig); err != nil {
		return err
	}

	_, err = globallock.With[struct{}](cmd.Context(), func() (struct{}, error) {
		if err := local.MoveIn(dstLocalName, tmpVMDir); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})

	return err
}
