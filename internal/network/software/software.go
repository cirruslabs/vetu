//go:build linux

package software

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/afpacket"
	"github.com/cirruslabs/vetu/internal/network/software/dhcp"
	"github.com/cirruslabs/vetu/internal/network/software/gvisor"
	"github.com/cirruslabs/vetu/internal/network/subnetfinder"
	"github.com/cirruslabs/vetu/internal/tuntap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"time"
)

var ErrInitFailed = errors.New("failed to initialize software networking")

type Network struct {
	tapFile *os.File
	ctx     context.Context
	cancel  context.CancelFunc
}

func New(vmHardwareAddr net.HardwareAddr) (*Network, error) {
	// Create a TAP interface for Cloud Hypervisor
	vmInterfaceName, vmTapFile, err := tuntap.CreateTAP("vetu%d", unix.IFF_VNET_HDR)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create a TAP interface: %v", ErrInitFailed, err)
	}

	vmLink, err := netlink.LinkByName(vmInterfaceName)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to find the TAP interface %q that we've just created: %v",
			ErrInitFailed, vmInterfaceName, err)
	}

	if err := netlink.LinkSetUp(vmLink); err != nil {
		return nil, fmt.Errorf("%w: failed to bring the TAP interface %q up: %v",
			ErrInitFailed, vmInterfaceName, err)
	}

	// Find an available subnet to use
	gatewayIP, vmIP, hostIP, network, err := subnetfinder.FindAvailableSubnet(29)
	if err != nil {
		return nil, err
	}

	// Work around systemd-udevd(8) imposing its own random MAC-address on the interface[1]
	// shortly after we create it, which results in the removal of our static neighbor.
	//
	// [1]: https://github.com/systemd/systemd/issues/21185
	time.Sleep(100 * time.Millisecond)

	// Add a permanent neighbor so that "vetu ip" would work
	if err := netlink.NeighAdd(&netlink.Neigh{
		LinkIndex:    vmLink.Attrs().Index,
		IP:           vmIP,
		HardwareAddr: vmHardwareAddr,
		State:        netlink.NUD_PERMANENT,
	}); err != nil {
		return nil, fmt.Errorf("%w: failed to add a permanent neighbor %s -> %s on an interface %s: %v",
			ErrInitFailed, vmIP, vmHardwareAddr, vmLink.Attrs().Name, err)
	}

	// Add an address so that we would be able to connect
	// to the VM by using an IP address returned by "vetu ip"
	if err := netlink.AddrAdd(vmLink, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   hostIP,
			Mask: network.Mask,
		},
	}); err != nil {
		return nil, fmt.Errorf("%w: failed to assign address %s to an interface %q: %v",
			ErrInitFailed, hostIP, vmLink.Attrs().Name, err)
	}

	rawSocketFD, err := afpacket.RawSocket(vmLink.Attrs().Index)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create a raw socket for the interface %q: %v",
			ErrInitFailed, vmLink.Attrs().Name, err)
	}

	gvisor, err := gvisor.New(rawSocketFD, gatewayIP, network)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitFailed, err)
	}

	dhcp, err := dhcp.New(gvisor.Stack(), gatewayIP, vmIP)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitFailed, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := gvisor.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := dhcp.Run(context.Background()); err != nil {
			panic(err)
		}
	}()

	return &Network{
		tapFile: vmTapFile,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (network *Network) SupportsOffload() bool {
	return false
}

func (network *Network) Tap() *os.File {
	return network.tapFile
}

func (network *Network) Close() error {
	network.cancel()

	return nil
}
