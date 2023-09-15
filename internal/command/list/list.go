package list

import (
	"fmt"
	"github.com/cirruslabs/nutmeg/internal/storage/local"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

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
	localVMs, err := local.List()
	if err != nil {
		return err
	}

	table := uitable.New()

	table.AddRow("Source", "Name")

	for _, localVM := range localVMs {
		table.AddRow("local", localVM.Path())
	}

	fmt.Println(table.String())

	return nil
}
