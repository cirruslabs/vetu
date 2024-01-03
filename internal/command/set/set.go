package set

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var cpu uint8
var memory uint16
var diskSize uint16

var ErrSet = errors.New("failed to set VM configuration")

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
	cmd.Flags().Uint16Var(&diskSize, "disk-size", 0, "resize the primary VMs disk "+
		"to the specified size in GB (note that the disk size can only be increased to avoid losing data)")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	name := args[0]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	// Open and lock VM directory (under a global lock) until the end of the "vetu set" execution
	vmDir, err := globallock.With(func() (*vmdirectory.VMDirectory, error) {
		vmDir, err := local.Open(localName)
		if err != nil {
			return nil, err
		}

		lock, err := vmDir.FileLock()
		if err != nil {
			return nil, err
		}

		if err := lock.Trylock(); err != nil {
			return nil, err
		}

		return vmDir, nil
	})
	if err != nil {
		return err
	}

	vmConfig, err := vmDir.Config()
	if err != nil {
		return err
	}

	if cpu != 0 {
		vmConfig.CPUCount = cpu
	}

	if memory != 0 {
		vmConfig.MemorySize = uint64(memory) * 1024 * 1024
	}

	if diskSize != 0 {
		if err := resizeDisk(vmDir, vmConfig); err != nil {
			return err
		}
	}

	return vmDir.SetConfig(vmConfig)
}

func resizeDisk(vmDir *vmdirectory.VMDirectory, vmConfig *vmconfig.VMConfig) error {
	if len(vmConfig.Disks) < 1 {
		return fmt.Errorf("%w: VM has no disks", ErrSet)
	}

	diskFile, err := os.OpenFile(filepath.Join(vmDir.Path(), vmConfig.Disks[0].Name), os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("%w: failed to open disk %s: %v", ErrSet, vmConfig.Disks[0].Name, err)
	}

	diskStat, err := diskFile.Stat()
	if err != nil {
		return fmt.Errorf("%w: failed to retrieve the size of disk %s: %v",
			ErrSet, vmConfig.Disks[0].Name, err)
	}

	desiredDiskSizeBytes := int64(diskSize) * humanize.GByte

	if actualDiskSizeBytes := diskStat.Size(); desiredDiskSizeBytes <= actualDiskSizeBytes {
		return fmt.Errorf("%w: new disk size of %s should be larger than the current disk size of %s",
			ErrSet, humanize.Bytes(uint64(desiredDiskSizeBytes)), humanize.Bytes(uint64(actualDiskSizeBytes)))
	}

	if err := diskFile.Truncate(desiredDiskSizeBytes); err != nil {
		return fmt.Errorf("%w: failed to truncate disk %s: %v", ErrSet, vmConfig.Disks[0].Name, err)
	}

	if err := diskFile.Close(); err != nil {
		return fmt.Errorf("%w: failed to close disk %s: %v", ErrSet, vmConfig.Disks[0].Name, err)
	}

	return nil
}
