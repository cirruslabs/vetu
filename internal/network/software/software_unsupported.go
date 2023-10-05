//go:build !linux

package software

import (
	"errors"
	"net"
	"os"
)

var ErrNotSupported = errors.New("software networking is not supported on this platform")

type Network struct{}

func New(vmHardwareAddr net.HardwareAddr) (*Network, error) {
	return nil, ErrNotSupported
}

func (network *Network) SupportsOffload() bool {
	return false
}

func (network *Network) Tap() *os.File {
	return nil
}

func (network *Network) Close() error {
	return nil
}
