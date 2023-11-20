package ip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"net"
	"time"
)

var wait uint16

var ErrIPNotFound = errors.New("VM's IP not found in the ARP cache, is the VM running?")

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Get VM's IP address",
		RunE:  runIP,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Uint16Var(&wait, "wait", 0,
		"number of seconds to wait for a potential VM booting")

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

	hardwareAddr := vmDir.Config().MACAddress.HardwareAddr

	subCtx, cancel := context.WithTimeout(cmd.Context(), time.Duration(wait)*time.Second)
	defer cancel()

	err = retry.Do(func() error {
		ip, err := arpTableLookup(hardwareAddr)
		if err != nil {
			return err
		}

		fmt.Println(ip)

		return nil
	}, retry.Context(subCtx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(1*time.Second),
		retry.LastErrorOnly(true),
	)
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrIPNotFound
	}

	return err
}

func arpTableLookup(hardwareAddr net.HardwareAddr) (string, error) {
	neighbors, err := netlink.NeighList(0, 0)
	if err != nil {
		return "", err
	}

	for _, neigh := range neighbors {
		if bytes.Equal(neigh.HardwareAddr, hardwareAddr) {
			return neigh.IP.String(), nil
		}
	}

	return "", ErrIPNotFound
}
