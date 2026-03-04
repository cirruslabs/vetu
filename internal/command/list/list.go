package list

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/gosuri/uitable"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type Source struct {
	Name     string
	ListFunc func() ([]lo.Tuple2[string, *vmdirectory.VMDirectory], error)
}

type VMInfo struct {
	Name    string
	Source  string
	State   string
	Running bool
	Disk    uint64
}

var source string
var quiet bool
var format string

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
	cmd.Flags().StringVar(&format, "format", "text", "output format, either \"text\" or \"json\"")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	var desiredSources []Source

	// Support --source
	switch source {
	case "local":
		desiredSources = append(desiredSources,
			Source{"local", local.List})
	case "oci":
		desiredSources = append(desiredSources,
			Source{"OCI", remote.List})
	case "":
		desiredSources = append(desiredSources,
			Source{"local", local.List},
			Source{"OCI", remote.List})
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

	// Retrieve VM infos from each desired source under a global lock
	var vmInfos []VMInfo

	_, err := globallock.With(cmd.Context(), func() (struct{}, error) {
		for _, desiredSource := range desiredSources {
			vmInfosLocal, err := vmInfosForSource(desiredSource)
			if err != nil {
				return struct{}{}, err
			}

			vmInfos = append(vmInfos, vmInfosLocal...)
		}

		return struct{}{}, nil
	})
	if err != nil {
		return err
	}

	// Render VM infos depending on the requested format
	switch format {
	case "text":
		table := uitable.New()

		table.AddRow("Source", "Name", "Disk", "State")

		for _, vmInfo := range vmInfos {
			table.AddRow(vmInfo.Source, vmInfo.Name, vmInfo.Disk, vmInfo.State)
		}

		fmt.Println(table.String())
	case "json":
		encoder := json.NewEncoder(os.Stdout)

		encoder.SetIndent("", "  ")

		if err := encoder.Encode(vmInfos); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot display VMs in an unsupported format %q", format)
	}

	return nil
}

func vmInfosForSource(source Source) ([]VMInfo, error) {
	var vmInfos []VMInfo

	vms, err := source.ListFunc()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		name, vmDir := lo.Unpack2(vm)

		diskSizeBytes, err := vmDir.Size()
		if err != nil {
			return nil, err
		}

		state := vmDir.State()

		vmInfos = append(vmInfos, VMInfo{
			Name:    name,
			Source:  source.Name,
			State:   string(state),
			Running: state == vmdirectory.StateRunning,
			Disk:    diskSizeBytes / humanize.GByte,
		})
	}

	return vmInfos, nil
}
