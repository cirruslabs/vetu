//nolint:predeclared // that's ok, we import it as deletepkg
package delete

import (
	"fmt"
	namepkg "github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a VM",
		RunE:  runDelete,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	var names []namepkg.Name

	for _, rawName := range args {
		name, err := namepkg.NewFromString(rawName)
		if err != nil {
			return err
		}

		names = append(names, name)
	}

	for _, name := range names {
		var err error

		switch typedName := name.(type) {
		case localname.LocalName:
			err = local.Delete(typedName)
		case remotename.RemoteName:
			err = remote.Delete(typedName)
		}

		if err != nil {
			fmt.Printf("failed to delete VM %q: %v\n", name, err)
		}
	}

	return nil
}
