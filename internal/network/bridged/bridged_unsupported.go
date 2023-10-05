//go:build !linux

package bridged

import (
	"errors"
	"os"
)

var ErrNotSupported = errors.New("bridged networking is not supported on this platform")

type Network struct{}

func New(bridgeName string) (*Network, error) {
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
