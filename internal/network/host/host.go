//go:build linux

package host

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cirruslabs/vetu/internal/network/subnetfinder"
	"github.com/cirruslabs/vetu/internal/tuntap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type Network struct {
	tapFile *os.File
	network net.IPNet
}

func New(vmHardwareAddr net.HardwareAddr, mtu int) (*Network, error) {
	// Create a TAP interface
	tapName, tapFile, err := tuntap.CreateTAP("vetu%d", unix.IFF_VNET_HDR)
	if err != nil {
		return nil, err
	}

	// Locate the TAP interface
	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the TAP interface %q that we've just created: %v",
			tapName, err)
	}

	// Set the MTU of the TAP interface
	if mtu != 0 {
		if err := netlink.LinkSetMTU(tapLink, mtu); err != nil {
			return nil, err
		}
	}

	// Bring the TAP interface up
	if err := netlink.LinkSetUp(tapLink); err != nil {
		return nil, fmt.Errorf("failed to bring the TAP interface %q up: %v", tapName, err)
	}

	// Find an available subnet to use
	hostIP, vmIP, _, network, err := subnetfinder.FindAvailableSubnet(29)
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
		LinkIndex:    tapLink.Attrs().Index,
		IP:           vmIP,
		HardwareAddr: vmHardwareAddr,
		State:        netlink.NUD_PERMANENT,
	}); err != nil {
		return nil, fmt.Errorf("failed to add a permanent neighbor %s -> %s on an interface %s: %v",
			vmIP, vmHardwareAddr, tapLink.Attrs().Name, err)
	}

	// Add an address to the TAP interface so that we would be able to
	// connect to the VM by using an IP address returned by "vetu ip"
	if err := netlink.AddrAdd(tapLink, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   hostIP,
			Mask: network.Mask,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to assign address %s to an interface %q: %v",
			hostIP, tapLink.Attrs().Name, err)
	}

	// Provide a DHCP service
	dhcp, err := NewDHCPServer(tapLink.Attrs().Name, hostIP, vmIP)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a DHCP server: %v", err)
	}

	go func() {
		if err := dhcp.Run(context.Background()); err != nil {
			panic(err)
		}
	}()

	return &Network{
		tapFile: tapFile,
		network: network,
	}, nil
}

func (network *Network) SupportsOffload() bool {
	return true
}

func (network *Network) Tap() *os.File {
	return network.tapFile
}

func (network *Network) Close() error {
	var srcFilter netlink.ConntrackFilter
	if err := srcFilter.AddIPNet(netlink.ConntrackOrigSrcIP, &network.network); err != nil {
		return fmt.Errorf("failed to add IP network to an source conntrack filter: %w", err)
	}

	var dstFilter netlink.ConntrackFilter
	if err := srcFilter.AddIPNet(netlink.ConntrackOrigDstIP, &network.network); err != nil {
		return fmt.Errorf("failed to add IP network to an destination conntrack filter: %w", err)
	}

	_, err := netlink.ConntrackDeleteFilters(netlink.ConntrackTable, unix.AF_INET, &srcFilter, &dstFilter)
	if err != nil {
		return fmt.Errorf("failed to delete conntrack entries: %w", err)
	}

	return nil
}
