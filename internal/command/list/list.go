package list

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/gosuri/uitable"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type desiredSource struct {
	Name     string
	ListFunc func() ([]lo.Tuple2[string, *vmdirectory.VMDirectory], error)
}

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
	var desiredSources []desiredSource

	// Support --source
	switch source {
	case "local":
		desiredSources = append(desiredSources,
			desiredSource{"local", local.List})
	case "oci":
		desiredSources = append(desiredSources,
			desiredSource{"oci", remote.List})
	case "":
		desiredSources = append(desiredSources,
			desiredSource{"local", local.List},
			desiredSource{"oci", remote.List})
	default:
		return fmt.Errorf("cannot display VMs from an unsupported source %q", source)
	}

	// Support -q/--quiet
	if quiet {
		for _, list := range desiredSources {
			vms, err := list.ListFunc()
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

	// Retrieve VMs metadata under a global lock
	_, err := globallock.With(cmd.Context(), func() (struct{}, error) {
		for _, desiredSource := range desiredSources {
			if err := addVMsToTable(table, desiredSource); err != nil {
				return struct{}{}, err
			}
		}

		return struct{}{}, nil
	})
	if err != nil {
		return err
	}

	fmt.Println(table.String())

	return nil
}

func addVMsToTable(table *uitable.Table, desiredSource desiredSource) error {
	vms, err := desiredSource.ListFunc()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		name, vmDir := lo.Unpack2(vm)

		size, err := vmDir.Size()
		if err != nil {
			return err
		}

		table.AddRow(desiredSource.Name, name, humanize.Bytes(size), vmDir.State())
	}

	return nil
}
