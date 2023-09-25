package ip

import (
	"bytes"
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Get VM's IP address",
		RunE:  runIP,
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}

func runIP(cmd *cobra.Command, args []string) error {
	name := args[0]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	// Look up ARP table
	neighbors, err := netlink.NeighList(0, 0)
	if err != nil {
		return nil
	}

	for _, neigh := range neighbors {
		if bytes.Equal(neigh.HardwareAddr, vmDir.Config().MACAddress.HardwareAddr) {
			fmt.Println(neigh.IP.String())

			return nil
		}
	}

	return fmt.Errorf("VM's IP not found in the ARP cache, is the VM running?")
}
