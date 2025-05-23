package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/externalcommand/cloudhypervisor"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/network"
	"github.com/cirruslabs/vetu/internal/network/bridged"
	"github.com/cirruslabs/vetu/internal/network/host"
	"github.com/cirruslabs/vetu/internal/network/software"
	"github.com/cirruslabs/vetu/internal/pidlock"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var netBridged string
var netHost bool
var devices []string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a VM",
		RunE:  runRun,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&netBridged, "net-bridged", "", "specify a bridge interface "+
		"to attach the VM to instead of using the software TCP/IP stack by default")
	cmd.Flags().BoolVar(&netHost, "net-host", false, "use host-networking "+
		"(assigns the first available /30 subnet from the private IPv4 address space to the "+
		"\"vetu*\" interface and serves it using the built-in DHCP server to the VM)")
	cmd.Flags().StringArrayVar(&devices, "device", []string{},
		"direct device assignment `parameters` to pass to the Cloud Hypervisor command, can be "+
			"repeated multiple times to attach multiple devices (e.g. "+
			"--device=\"path=/sys/bus/pci/devices/0000:01:00.0/,iommu=on\")")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Only local VMs can be run
	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	// Open and lock VM directory (under a global lock) until the end of the "vetu run" execution
	vmDir, err := globallock.With(cmd.Context(), func() (*vmdirectory.VMDirectory, error) {
		vmDir, err := local.Open(localName)
		if err != nil {
			return nil, err
		}

		lock, err := vmDir.FileLock(filelock.LockExclusive)
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

	// Acquire a lock after reading the config[1]
	//
	//nolint:lll
	// [1]: https://github.com/cirruslabs/tart/blob/8c011623be2ed8254cd91b15c336c2fff2b6f9be/Sources/tart/Commands/Run.swift#L209-L220
	lock, err := pidlock.New(vmDir.ConfigPath())
	if err != nil {
		return err
	}
	if err := lock.Trylock(); err != nil {
		return fmt.Errorf("VM %q is already running", name)
	}

	// Validate VM's architecture
	if vmConfig.Arch != runtime.GOARCH {
		return fmt.Errorf("this VM is built to run on %q, but you're running %q",
			vmConfig.Arch, runtime.GOARCH)
	}

	// Initialize network
	network, err := globallock.With(cmd.Context(), func() (network.Network, error) {
		switch {
		case netBridged != "":
			return bridged.New(netBridged)
		case netHost:
			return host.New(vmConfig.MACAddress.HardwareAddr)
		default:
			return software.New(vmConfig.MACAddress.HardwareAddr)
		}
	})
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

	// Devices
	for _, device := range devices {
		hvArgs = append(hvArgs, "--device", device)
	}

	// Reduce VirtIO IOMMU address width from 64 to 39 bits
	// to avoid Cloud Hypervisor exists due to failed DMA
	// mappings on amd64[1].
	//
	// [1]: https://github.com/cloud-hypervisor/cloud-hypervisor/pull/6900
	if runtime.GOARCH == "amd64" && len(devices) != 0 {
		hvArgs = append(hvArgs, "--platform", "iommu_address_width=39")
	}

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

	// Graceful Cloud Hypervisor termination[1]
	//
	// [1]: https://www.cloudhypervisor.org/blog/cloud-hypervisor-v0.11.0-released/#sigtermsigint-interrupt-signal-handling
	hv.Cancel = func() error {
		return hv.Process.Signal(unix.SIGTERM)
	}

	// If Cancel() fails to terminate the Cloud Hypervisor for some reason,
	// ensure that it will eventually be killed after some time.
	hv.WaitDelay = 30 * time.Second

	if err := hv.Run(); err != nil {
		// Context cancellation is not an error
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	}

	return nil
}
