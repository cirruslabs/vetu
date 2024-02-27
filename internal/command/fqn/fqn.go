package fqn

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
	"path/filepath"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fqn",
		Short:  "Get a fully-qualified VM name",
		RunE:   runFQN,
		Args:   cobra.ExactArgs(1),
		Hidden: true,
	}

	return cmd
}

func runFQN(cmd *cobra.Command, args []string) error {
	name, err := name.NewFromString(args[0])
	if err != nil {
		return err
	}

	switch typedSrcName := name.(type) {
	case localname.LocalName:
		fmt.Println(typedSrcName.String())
	case remotename.RemoteName:
		resolvedPath, err := remote.PathForResolved(typedSrcName)
		if err != nil {
			return err
		}

		typedSrcName.Tag = ""
		typedSrcName.Digest = digest.Digest(filepath.Base(resolvedPath))

		fmt.Println(typedSrcName.String())
	}

	return nil
}
