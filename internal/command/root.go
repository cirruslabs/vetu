package command

import (
	"github.com/cirruslabs/vetu/internal/command/clone"
	"github.com/cirruslabs/vetu/internal/command/create"
	deletepkg "github.com/cirruslabs/vetu/internal/command/delete"
	"github.com/cirruslabs/vetu/internal/command/ip"
	"github.com/cirruslabs/vetu/internal/command/list"
	"github.com/cirruslabs/vetu/internal/command/run"
	"github.com/cirruslabs/vetu/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "vetu",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.FullVersion,
	}

	cmd.AddCommand(
		create.NewCommand(),
		clone.NewCommand(),
		run.NewCommand(),
		list.NewCommand(),
		ip.NewCommand(),
		deletepkg.NewCommand(),
	)

	return cmd
}
