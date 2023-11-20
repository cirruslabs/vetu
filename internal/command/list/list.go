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

var source string
var quiet bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VMs",
		RunE:  runList,
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().StringVar(&source, "source", "",
		"only display VMs from the specified source (e.g. --source local or --source oci)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "only display VM names")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	desiredSources := map[string]listFunc{}

	// Support --source
	switch source {
	case "local":
		desiredSources["local"] = local.List
	case "oci":
		desiredSources["oci"] = remote.List
	case "":
		desiredSources["local"] = local.List
		desiredSources["oci"] = remote.List
	default:
		return fmt.Errorf("cannot display VMs from an unsupported source %q", source)
	}

	// Support -q/--quiet
	if quiet {
		for _, list := range desiredSources {
			vms, err := list()
			if err != nil {
				return err
			}

			for _, vm := range vms {
				name, _ := lo.Unpack2(vm)

				fmt.Println(name)
			}
		}

		return nil
	}

	table := uitable.New()

	table.AddRow("Source", "Name", "Size", "State")

	for source, list := range desiredSources {
		if err := addVMsToTable(table, source, list); err != nil {
			return err
		}
	}

	fmt.Println(table.String())

	return nil
}

func addVMsToTable(table *uitable.Table, source string, list listFunc) error {
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

		table.AddRow(source, name, humanize.IBytes(size), vmDir.State())
	}

	return nil
}
