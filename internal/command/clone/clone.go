package clone

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone NAME",
		Short: "Clone a VM",
		RunE:  runClone,
		Args:  cobra.ExactArgs(2),
	}

	return cmd
}

func runClone(cmd *cobra.Command, args []string) error {
	srcName := args[0]
	dstName := args[1]

	srcLocalName, err := localname.NewFromString(srcName)
	if err != nil {
		return err
	}

	dstLocalName, err := localname.NewFromString(dstName)
	if err != nil {
		return err
	}

	if local.Exists(dstLocalName) {
		return fmt.Errorf("VM %q already exists", dstLocalName)
	}

	srcPath, err := local.PathFor(srcLocalName)
	if err != nil {
		return err
	}

	dstPath, err := local.PathFor(dstLocalName)
	if err != nil {
		return err
	}

	return temporary.AtomicallyCopyThrough(srcPath, dstPath)
}
