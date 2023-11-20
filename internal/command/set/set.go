package set

import (
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
)

var cpu uint8
var memory uint16

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Modify VM's configuration",
		RunE:  runSet,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Uint8Var(&cpu, "cpu", 2, "number of VM CPUs to use for the VM")
	cmd.Flags().Uint16Var(&memory, "memory", 4096, "amount of memory to use "+
		"for the VM in MiB (mebibytes)")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	name := args[0]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	vmConfig := vmDir.Config()

	if cpu != 0 {
		vmConfig.CPUCount = cpu
	}

	if memory != 0 {
		vmConfig.MemorySize = uint64(memory) * 1024 * 1024
	}

	return vmDir.SetConfig(&vmConfig)
}
