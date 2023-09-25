package software

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/afpacket"
	"github.com/cirruslabs/vetu/internal/externalcommand/passt"
	"github.com/cirruslabs/vetu/internal/randommac"
	"github.com/cirruslabs/vetu/internal/tuntap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"time"
)

type Network struct {
	tapFile *os.File
}

func New(ctx context.Context, vmHardwareAddr net.HardwareAddr) (*Network, error) {
	// Create a TAP interface for Cloud Hypervisor
	vmInterfaceName, vmTapFile, err := tuntap.CreateTAP("vetu%d", unix.IFF_VNET_HDR)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a TAP interface for Cloud Hypervisor: %v", err)
	}

	vmLink, err := netlink.LinkByName(vmInterfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the TAP interface for Cloud Hypervisor that we've just "+
			"created: %v", err)
	}

	if err := netlink.LinkSetUp(vmLink); err != nil {
		return nil, fmt.Errorf("failed to bring the TAP interface for Cloud Hypervisor: %v", err)
	}

	// Find an available subnet to use
	gatewayIP, vmIP, hostIP, network, err := FindAvailableSubnet(29)
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
		return nil, fmt.Errorf("failed to add a permanent neighbor: %v", err)
	}

	// Add an address so that we would be able to connect
	// to the VM by using an IP address returned by "vetu ip"
	if err := netlink.AddrAdd(vmLink, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   hostIP,
			Mask: network.GetNetworkMask().Bytes(),
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to assign address to a bridge interface: %v", err)
	}

	rawSocketFile, err := afpacket.RawSocket(vmLink.Attrs().Index)
	if err != nil {
		return nil, err
	}

	// Launch passt
	passtHardwareAddr, err := randommac.UnicastAndLocallyAdministered()
	if err != nil {
		return nil, fmt.Errorf("failed to create random MAC-address for passt")
	}

	passtCmd, err := passt.Passt(ctx, "--foreground", "--address", vmIP.String(),
		"--netmask", network.GetNetworkMask().String(), "--gateway", gatewayIP.String(),
		"--mac-addr", passtHardwareAddr.String(), "-4", "--mtu", "1500", "--tap-fd", "3")
	if err != nil {
		return nil, err
	}

	passtCmd.Stderr = os.Stderr
	passtCmd.Stdout = os.Stdout

	passtCmd.ExtraFiles = []*os.File{
		rawSocketFile,
	}

	if err := passtCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run passt: %v", err)
	}

	return &Network{
		tapFile: vmTapFile,
	}, nil
}

func (network *Network) Tap() *os.File {
	return network.tapFile
}

func (network *Network) Close() error {
	return nil
}
