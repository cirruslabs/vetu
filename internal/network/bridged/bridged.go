//go:build linux

package bridged

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/tuntap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"os"
)

type Network struct {
	tapFile *os.File
}

func New(bridgeName string) (*Network, error) {
	// Locate the bridge
	bridgeLink, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil, fmt.Errorf("bridge %q not found: %v", bridgeName, err)
	}

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

	// Bring the TAP interface up
	if err := netlink.LinkSetUp(tapLink); err != nil {
		return nil, fmt.Errorf("failed to bring the TAP interface %q up: %v", tapName, err)
	}

	// Attach the TAP interface to the bridge
	if err := netlink.LinkSetMaster(tapLink, bridgeLink); err != nil {
		return nil, fmt.Errorf("failed to attach the TAP interface %q to the bridge %q: %v",
			tapName, bridgeName, err)
	}

	return &Network{
		tapFile,
	}, nil
}

func (network *Network) SupportsOffload() bool {
	return true
}

func (network *Network) Tap() *os.File {
	return network.tapFile
}

func (network *Network) Close() error {
	return nil
}
