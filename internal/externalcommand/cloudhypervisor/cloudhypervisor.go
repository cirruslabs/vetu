package cloudhypervisor

import (
	"context"
	"os/exec"
)

const binaryName = "cloud-hypervisor"

func CloudHypervisor(ctx context.Context, args ...string) (*exec.Cmd, error) {
	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		return nil, err
	}

	return exec.CommandContext(ctx, binaryPath, args...), nil
}
