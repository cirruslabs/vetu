package software

import (
	"context"
	"fmt"
	"git.sr.ht/~jamesponddotco/acopw-go"
	"github.com/cirruslabs/nutmeg/internal/externalcommand/passt"
	"github.com/cirruslabs/nutmeg/internal/randommac"
	"github.com/cirruslabs/nutmeg/internal/tuntap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"time"
)

type Network struct {
	tapFile    *os.File
	bridgeName string
}

func New(ctx context.Context, vmHardwareAddr net.HardwareAddr) (*Network, error) {
	// Generate interface names that we'll use for this network instance
	//
	// Note that the maximum interface name is limited to 15 characters in Linux.
	randomComponent, err := (&acopw.Random{
		Length:     5,
		UseLower:   true,
		UseUpper:   false,
		UseNumbers: true,
		UseSymbols: false,
	}).Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random component: %v", err)
	}

	bridgeInterfaceName := fmt.Sprintf("nutmeg-%s-br", randomComponent)
	passtInterfaceName := fmt.Sprintf("nutmeg-%s-ps", randomComponent)
	vmInterfafceName := fmt.Sprintf("nutmeg-%s-vm", randomComponent)

	// Create a TAP interface for passt
	_, passtTapFile, err := tuntap.CreateTAP(passtInterfaceName, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a TAP interface for passt: %v", err)
	}

	passtLink, err := netlink.LinkByName(passtInterfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the TAP interface for passt that we've just "+
			"created: %v", err)
	}

	if err := netlink.LinkSetUp(passtLink); err != nil {
		return nil, fmt.Errorf("failed to bring the TAP interface for passt: %v", err)
	}

	// Create a TAP interface for Cloud Hypervisor
	_, vmTapFile, err := tuntap.CreateTAP(vmInterfafceName, unix.IFF_VNET_HDR)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a TAP interface for Cloud Hypervisor: %v", err)
	}

	vmLink, err := netlink.LinkByName(vmInterfafceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the TAP interface for Cloud Hypervisor that we've just "+
			"created: %v", err)
	}

	if err := netlink.LinkSetUp(vmLink); err != nil {
		return nil, fmt.Errorf("failed to bring the TAP interface for Cloud Hypervisor: %v", err)
	}

	// Create a bridge and add both of the TAP interfaces above to it
	bridgeLink, err := createBridgeWithLinks(ctx, bridgeInterfaceName, vmLink, passtLink)
	if err != nil {
		return nil, fmt.Errorf("failed to create bridge: %v", err)
	}

	// Find an available subnet to use
	gatewayIP, vmIP, hostIP, network, err := FindAvailableSubnet(29)
	if err != nil {
		return nil, err
	}

	// Add a permanent neighbor so that "nutmeg ip" would work
	if err := netlink.NeighAdd(&netlink.Neigh{
		LinkIndex:    bridgeLink.Attrs().Index,
		IP:           vmIP,
		HardwareAddr: vmHardwareAddr,
		State:        netlink.NUD_PERMANENT,
	}); err != nil {
		return nil, fmt.Errorf("failed to add a permanent neighbor: %v", err)
	}

	// Add an address so that we would be able to connect
	// to the VM by using an IP address returned by "nutmeg ip"
	if err := netlink.AddrAdd(bridgeLink, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   hostIP,
			Mask: network.GetNetworkMask().Bytes(),
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to assign address to a bridge interface: %v", err)
	}

	// Launch passt
	passtHardwareAddr, err := randommac.UnicastAndLocallyAdministered()
	if err != nil {
		return nil, fmt.Errorf("failed to create random MAC-address for passt")
	}

	passtCmd, err := passt.Passt(ctx, "--foreground", "--address", vmIP.String(),
		"--netmask", network.GetNetworkMask().String(), "--gateway", gatewayIP.String(),
		"--mac-addr", passtHardwareAddr.String(), "-4", "--mtu", "1500",
		"--fd", "3", "--fd-is-tap")
	if err != nil {
		return nil, err
	}

	passtCmd.Stderr = os.Stderr
	passtCmd.Stdout = os.Stdout

	passtCmd.ExtraFiles = []*os.File{
		passtTapFile,
	}

	if err := passtCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run passt: %v", err)
	}

	return &Network{
		tapFile:    vmTapFile,
		bridgeName: bridgeInterfaceName,
	}, nil
}

func createBridgeWithLinks(ctx context.Context, name string, linksToAdd ...netlink.Link) (netlink.Link, error) {
	// Create a bridge
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = name

	bridgeLinkUncooked := &netlink.Bridge{
		LinkAttrs: linkAttrs,
	}

	if err := netlink.LinkAdd(bridgeLinkUncooked); err != nil {
		return nil, fmt.Errorf("could not add %s: %v", name, err)
	}

	bridgeLink, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	// Add interfaces
	for _, linkToAdd := range linksToAdd {
		if err := netlink.LinkSetMaster(linkToAdd, bridgeLink); err != nil {
			return nil, err
		}
	}

	// Bring the bridge up
	if err := netlink.LinkSetUp(bridgeLink); err != nil {
		return nil, err
	}

	// Wait for the bridge to become "up"
	for {
		bridgeLink, err = netlink.LinkByName(name)
		if err != nil {
			return nil, err
		}

		if (bridgeLink.Attrs().Flags & net.FlagUp) != 0 {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return bridgeLink, nil
}

func (network *Network) Tap() *os.File {
	return network.tapFile
}

func (network *Network) Close() error {
	// Remove bridge interface
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = network.bridgeName

	return netlink.LinkDel(&netlink.Bridge{
		LinkAttrs: linkAttrs,
	})
}
