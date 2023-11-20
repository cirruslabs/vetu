package stop

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"time"
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

	// Open VM's directory
	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	// Find the PID of the "vetu run" process running this VM
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

	// Try to gracefully terminate the VM
	_ = unix.Kill(int(pid), unix.SIGINT)

	gracefulTerminationCtx, gracefulTerminationCtxCancel := context.WithTimeout(cmd.Context(),
		time.Duration(timeout)*time.Second)
	defer gracefulTerminationCtxCancel()

	err = retry.Do(func() error {
		pid, err := lock.Pid()
		if err != nil {
			return err
		}

		if pid == 0 {
			return nil
		}

		return fmt.Errorf("VM is still running")
	}, retry.Context(gracefulTerminationCtx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(100*time.Millisecond),
	)
	if err != nil {
		// Forcefully terminate the VM
		return unix.Kill(int(pid), unix.SIGKILL)
	}

	return nil
}
