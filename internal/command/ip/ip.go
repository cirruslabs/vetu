package ip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/vetu/internal/globallock"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/vmconfig"
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

	// Open the VM directory and read its configuration under a global lock
	vmConfig, err := globallock.With(cmd.Context(), func() (*vmconfig.VMConfig, error) {
		vmDir, err := local.Open(localName)
		if err != nil {
			return nil, err
		}

		return vmDir.Config()
	})
	if err != nil {
		return err
	}

	hardwareAddr := vmConfig.MACAddress.HardwareAddr

	retryOpts := []retry.Option{
		retry.DelayType(retry.FixedDelay),
		retry.Delay(1 * time.Second),
		retry.LastErrorOnly(true),
	}

	if wait == 0 {
		retryOpts = append(retryOpts, retry.Context(cmd.Context()), retry.Attempts(1))
	} else {
		waitCtx, waitCtxCancel := context.WithTimeout(cmd.Context(), time.Duration(wait)*time.Second)
		defer waitCtxCancel()

		retryOpts = append(retryOpts, retry.Context(waitCtx), retry.Attempts(0),
			retry.Delay(time.Second), retry.DelayType(retry.FixedDelay))
	}

	err = retry.Do(func() error {
		ip, err := arpTableLookup(hardwareAddr)
		if err != nil {
			return err
		}

		fmt.Println(ip)

		return nil
	}, retryOpts...)
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
