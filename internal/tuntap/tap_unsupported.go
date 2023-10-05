//go:build !linux

package tuntap

import (
	"errors"
	"os"
)

var ErrNotSupported = errors.New("TAP device is not supported on this platform")

func CreateTAP(name string, additionalFlags uint16) (string, *os.File, error) {
	return "", nil, ErrNotSupported
}
