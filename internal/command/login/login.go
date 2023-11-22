package login

import (
	"bufio"
	"fmt"
	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
	"syscall"
)

var username string
var passwordStdin bool
var noValidate bool

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login REGISTRY",
		Short: "Login to a registry",
		RunE:  runLogin,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&username, "username", "",
		"use the specified username instead of asking it interactively "+
			"(requires --password-stdin)")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false,
		"receive the password from the standard input instead of asking it interactively "+
			"(requires --username)")
	cmd.Flags().BoolVar(&noValidate, "no-validate", false,
		"skip validation of the OCI registry's credentials before logging-in")

	return cmd
}

func runLogin(cmd *cobra.Command, args []string) error {
	registry := args[0]

	// Retrieve credentials
	username, password, err := retrieveCredentials()
	if err != nil {
		return err
	}

	if !noValidate {
		// Create an OCI registry client that only has the provided credentials
		client := regclient.New(regclient.WithConfigHost(config.Host{
			Name:     registry,
			Hostname: registry,
			User:     username,
			Pass:     password,
		}))

		// Create a reference that only has the Registry field set
		reference, err := ref.NewHost(registry)
		if err != nil {
			return err
		}

		// Validate credentials
		_, err = client.Ping(cmd.Context(), reference)
		if err != nil {
			return err
		}
	}

	// Store credentials
	configFile, err := dockerconfig.Load("")
	if err != nil {
		return err
	}

	configFile.AuthConfigs[registry] = types.AuthConfig{
		Username: username,
		Password: password,
	}

	return configFile.Save()
}

func retrieveCredentials() (string, string, error) {
	switch {
	case username != "" && passwordStdin:
		// Both --username and --password-stdin were provided,
		// read the password from the standard input
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", "", err
		}

		return strings.TrimSpace(username), strings.TrimSpace(string(password)), nil
	case username == "" && !passwordStdin:
		// No --username nor --password-stdin were provided,
		// read the credentials interactively
		fmt.Print("Username: ")

		reader := bufio.NewReader(os.Stdin)

		username, err := reader.ReadString('\n')
		if err != nil {
			return "", "", err
		}

		fmt.Print("Password: ")

		password, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return "", "", err
		}

		return strings.TrimSpace(username), strings.TrimSpace(string(password)), nil
	default:
		return "", "", fmt.Errorf("please provide both --username and --password-stdin")
	}
}
