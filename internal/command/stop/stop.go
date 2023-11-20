package stop

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var timeout uint16

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a VM",
		RunE:  runStop,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Uint16Var(&timeout, "timeout", 30,
		"seconds to wait for graceful termination before forcefully terminating the VM")

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	name := args[0]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	lock, err := filelock.New(vmDir.ConfigPath())
	if err != nil {
		return err
	}

	pid, err := lock.Pid()
	if err != nil {
		return err
	}

	if pid == 0 {
		return fmt.Errorf("VM %q is not running", name)
	}

	return unix.Kill(int(pid), unix.SIGKILL)
}
