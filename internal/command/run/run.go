package run

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/externalcommand/cloudhypervisor"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/network"
	"github.com/cirruslabs/vetu/internal/network/bridged"
	"github.com/cirruslabs/vetu/internal/network/software"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var netBridged string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a VM",
		RunE:  runRun,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&netBridged, "net-bridged", "", "specify a bridge interface "+
		"to attach the VM to instead of using the software TCP/IP stack by default")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Only local VMs can be run
	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	// Acquire a lock
	lock, err := filelock.New(vmDir.ConfigPath())
	if err != nil {
		return fmt.Errorf("VM %q is already running", name)
	}
	defer func() {
		_ = lock.Unlock()
	}()

	vmConfig := vmDir.Config()

	// Validate VM's architecture
	if vmConfig.Arch != runtime.GOARCH {
		return fmt.Errorf("this VM is built to run on %q, but you're running %q",
			vmConfig.Arch, runtime.GOARCH)
	}

	// Initialize network
	var network network.Network

	switch {
	case netBridged != "":
		network, err = bridged.New(netBridged)
	default:
		network, err = software.New(vmConfig.MACAddress.HardwareAddr)
	}
	if err != nil {
		return fmt.Errorf("failed to initialize VM's network: %v", err)
	}
	defer network.Close()

	// Kernel
	hvArgs := []string{"--console", "pty", "--serial", "tty", "--kernel", vmDir.KernelPath()}

	// Initramfs
	_, err = os.Stat(vmDir.InitramfsPath())
	if err == nil {
		hvArgs = append(hvArgs, "--initramfs", vmDir.InitramfsPath())
	}

	// Command-line
	if vmConfig.Cmdline != "" {
		hvArgs = append(hvArgs, "--cmdline", vmConfig.Cmdline)
	}

	// Disks
	diskArguments := lo.Map(vmConfig.Disks, func(disk vmconfig.Disk, index int) string {
		path := filepath.Join(vmDir.Path(), disk.Name)
		return fmt.Sprintf("path=%s", path)
	})
	if len(diskArguments) != 0 {
		hvArgs = append(hvArgs, "--disk")
		hvArgs = append(hvArgs, diskArguments...)
	}

	// CPU and memory
	if cpuCount := vmConfig.CPUCount; cpuCount != 0 {
		hvArgs = append(hvArgs, "--cpus", fmt.Sprintf("boot=%d", cpuCount))
	}

	if memorySize := vmConfig.MemorySize; memorySize != 0 {
		hvArgs = append(hvArgs, "--memory", fmt.Sprintf("size=%d", memorySize))
	}

	// Networking
	netOpts := []string{"fd=3", fmt.Sprintf("mac=%s", vmConfig.MACAddress)}

	if !network.SupportsOffload() {
		netOpts = append(netOpts, "offload_tso=off", "offload_ufo=off", "offload_csum=off")
	}

	hvArgs = append(hvArgs, "--net", strings.Join(netOpts, ","))

	hv, err := cloudhypervisor.CloudHypervisor(cmd.Context(), hvArgs...)
	if err != nil {
		return err
	}

	// Attach network's TAP interface
	//
	// The FD for the first ExtraFiles entry will always be 3,
	// as per ExtraFiles documentation: "If non-nil, entry i
	// becomes file descriptor 3+i".
	hv.ExtraFiles = []*os.File{
		network.Tap(),
	}

	hv.Stdout = os.Stdout
	hv.Stderr = os.Stderr
	hv.Stdin = os.Stdin

	return hv.Run()
}
