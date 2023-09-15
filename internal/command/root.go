package command

import (
	"github.com/cirruslabs/nutmeg/internal/command/clone"
	"github.com/cirruslabs/nutmeg/internal/command/create"
	deletepkg "github.com/cirruslabs/nutmeg/internal/command/delete"
	"github.com/cirruslabs/nutmeg/internal/command/ip"
	"github.com/cirruslabs/nutmeg/internal/command/list"
	"github.com/cirruslabs/nutmeg/internal/command/run"
	"github.com/cirruslabs/nutmeg/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "nutmeg",
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
