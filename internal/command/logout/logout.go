package logout

import (
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout REGISTRY",
		Short: "Logout from a registry",
		RunE:  runLogout,
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}

func runLogout(cmd *cobra.Command, args []string) error {
	registry := args[0]

	configFile, err := config.Load("")
	if err != nil {
		return err
	}

	delete(configFile.AuthConfigs, registry)

	return configFile.Save()
}
