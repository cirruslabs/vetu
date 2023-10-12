package clone

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone NAME LOCAL_NAME",
		Short: "Clone a VM",
		RunE:  runClone,
		Args:  cobra.ExactArgs(2),
	}

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

	// Open the source VM directory
	var srcVMDir *vmdirectory.VMDirectory

	switch typedSrcName := srcName.(type) {
	case localname.LocalName:
		srcVMDir, err = local.Open(typedSrcName)
	case remotename.RemoteName:
		srcVMDir, err = remote.Open(typedSrcName)
	}
	if err != nil {
		return err
	}

	// Ensure the target VM directory does not exist
	if local.Exists(dstLocalName) {
		return fmt.Errorf("VM %q already exists", dstLocalName)
	}

	// Retrieve a path for the target VM directory
	dstPath, err := local.PathFor(dstLocalName)
	if err != nil {
		return err
	}

	return temporary.AtomicallyCopyThrough(srcVMDir.Path(), dstPath)
}
