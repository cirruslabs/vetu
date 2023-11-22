package create

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/randommac"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"path/filepath"
)

var kernel string
var initramfs string
var cmdline string
var disks []string
var cpu uint8
var memory uint16

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a VM",
		RunE:  runCreate,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&kernel, "kernel", "", "path to a kernel file to use "+
		"for the new VM (will be copied to the VM's directory)")
	cmd.Flags().StringVar(&initramfs, "initramfs", "", "path to an initramfs file to use "+
		"for the new VM (will be copied to the VM's directory")
	cmd.Flags().StringVar(&cmdline, "cmdline", "", "kernel command-line parameters to use "+
		"when booting the new VM")
	cmd.Flags().StringArrayVar(&disks, "disk", []string{}, "path to a disk file to use "+
		"when booting the new VM (can be specified multiple times, will be copied to the VM's directory)")
	cmd.Flags().Uint8Var(&cpu, "cpu", 2, "number of VM CPUs to use "+
		"for the new VM")
	cmd.Flags().Uint16Var(&memory, "memory", 4096, "amount of memory to use "+
		"for the new VM in MiB (mebibytes)")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	if local.Exists(localName) {
		return fmt.Errorf("VM %q already exists", localName.String())
	}

	vmDir, err := temporary.Create()
	if err != nil {
		return err
	}

	vmConfig := vmDir.Config()

	// Kernel
	if kernel != "" {
		if err := cp.Copy(kernel, vmDir.KernelPath()); err != nil {
			return fmt.Errorf("failed to copy kernel to the VM's directory: %v", err)
		}
	} else {
		return fmt.Errorf("please specify a kernel using --kernel, otherwise the VM will not be bootable")
	}

	// Initramfs
	if initramfs != "" {
		if err := cp.Copy(initramfs, vmDir.InitramfsPath()); err != nil {
			return fmt.Errorf("failed to copy initramfs to the VM's directory: %v", err)
		}
	}

	// Command-line
	if cmdline != "" {
		vmConfig.Cmdline = cmdline
	}

	// Disks
	for _, disk := range disks {
		diskName := filepath.Base(disk)

		if err := cp.Copy(disk, filepath.Join(vmDir.Path(), diskName)); err != nil {
			return fmt.Errorf("failed to copy disk %q to the VM's directory: %v", diskName, err)
		}

		vmConfig.Disks = append(vmConfig.Disks, vmconfig.Disk{
			Name: diskName,
		})
	}

	// CPU and memory
	vmConfig.CPUCount = cpu
	vmConfig.MemorySize = uint64(memory) * 1024 * 1024

	// MAC address
	randomMAC, err := randommac.UnicastAndLocallyAdministered()
	if err != nil {
		return err
	}
	vmConfig.MACAddress.HardwareAddr = randomMAC

	if err := vmDir.SetConfig(&vmConfig); err != nil {
		return err
	}

	return local.MoveIn(localName, vmDir)
}
