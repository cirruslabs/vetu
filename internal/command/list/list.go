package list

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/gosuri/uitable"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type listFunc func() ([]lo.Tuple2[string, *vmdirectory.VMDirectory], error)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VMs",
		RunE:  runList,
		Args:  cobra.ExactArgs(0),
	}

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	table := uitable.New()

	table.AddRow("Source", "Name", "Size")

	if err := addVMs(local.List, "local", table); err != nil {
		return err
	}

	if err := addVMs(remote.List, "oci", table); err != nil {
		return err
	}

	fmt.Println(table.String())

	return nil
}

func addVMs(list listFunc, source string, table *uitable.Table) error {
	vms, err := list()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		name, vmDir := lo.Unpack2(vm)

		size, err := vmDir.Size()
		if err != nil {
			return err
		}

		table.AddRow(source, name, humanize.IBytes(size))
	}

	return nil
}
