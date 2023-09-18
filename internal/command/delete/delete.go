//nolint:predeclared // that's ok, we import it as deletepkg
package delete

import (
	"fmt"
	"github.com/cirruslabs/nutmeg/internal/name/localname"
	"github.com/cirruslabs/nutmeg/internal/storage/local"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a VM",
		RunE:  runDelete,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	var localNames []localname.LocalName

	for _, name := range args {
		localName, err := localname.NewFromString(name)
		if err != nil {
			return err
		}

		localNames = append(localNames, localName)
	}

	for _, localName := range localNames {
		if err := local.Delete(localName); err != nil {
			fmt.Printf("failed to delete VM %q: %v\n", localName, err)
		}
	}

	return nil
}
